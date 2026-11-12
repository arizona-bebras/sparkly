package main

import (
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/validate"
	"github.com/joho/godotenv"

	"sparkly/backend/internal/db"
	"sparkly/backend/internal/handlers"
	"sparkly/backend/internal/interceptors"
	"sparkly/backend/internal/msgq"
	"sparkly/backend/internal/repos"
	"sparkly/backend/internal/services"
	"sparkly/backend/proto/v1/backendv1connect"
)

func main() {
	env := struct {
		POSTGRES_HOST     string
		POSTGRES_DB       string
		POSTGRES_USER     string
		POSTGRES_PASSWORD string
		POSTGRES_PORT     string
		MAIN_PORT         string
		RABBITMQ_HOST     string
		RABBITMQ_PORT     string
	}{}

	readEnv := func() {
		err := godotenv.Load()
		if err != nil {
			log.Printf("failed to load .env file: %v", err)
		}
		env.POSTGRES_HOST, _ = os.LookupEnv("POSTGRES_HOST")
		env.POSTGRES_DB, _ = os.LookupEnv("POSTGRES_DB")
		env.POSTGRES_USER, _ = os.LookupEnv("POSTGRES_USER")
		env.POSTGRES_PASSWORD, _ = os.LookupEnv("POSTGRES_PASSWORD")
		env.POSTGRES_PORT, _ = os.LookupEnv("POSTGRES_PORT")
		env.MAIN_PORT, _ = os.LookupEnv("MAIN_PORT")
		if env.MAIN_PORT == "" {
			env.MAIN_PORT = "8080"
		}
	}
	readEnv()

	db, err := db.NewPostgresConn(db.DSN{
		Host:     env.POSTGRES_HOST,
		User:     env.POSTGRES_USER,
		Password: env.POSTGRES_PASSWORD,
		DBName:   env.POSTGRES_DB,
		Port:     env.POSTGRES_PORT,
		SSLMode:  "disable",
		TimeZone: "Asia/Yekaterinburg",
	})
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}

	broker, err := msgq.NewMessageBroker(env.RABBITMQ_HOST, env.RABBITMQ_PORT)
	if err != nil {
		log.Fatalf("failed to init rabbitmq broker: %w", err)
	}

	jwksRepo, err := repos.NewJWKSRepo(db)
	if err != nil {
		log.Fatalf("failed to init keys repo: %v", err)
	}
	jwksService, err := services.NewJWKSService(jwksRepo)
	if err != nil {
		log.Fatalf("failed to initialize jwks: %v", err)
	}
	jwksHandler := handlers.NewJWKSHandler(jwksService)

	refreshTokenRepo := repos.NewRefreshTokenRepo(db)
	tokenRefresher := interceptors.NewTokenRefresher(
		jwksService,
		refreshTokenRepo,
	)

	server := connect.NewServer(
		validate.NewServerInterceptor(),
		interceptors.NewAuthInterceptor(tokenRefresher),
	)

	userRepo := repos.NewUserRepo(db)
	userService := services.NewUserService(userRepo, refreshTokenRepo, jwksService)
	userHandler := handlers.NewUserHandler(userService)
	backendv1connect.RegisterUserServiceHandler(server, userHandler)

	parkRepo := repos.NewParkRepo(db)
	parkService := services.NewParkService(parkRepo)
	groupsRepo := repos.NewGroupsRepo(db)
	groupsService := services.NewGroupsService(groupsRepo)
	parkHandler := handlers.NewParkHandler(parkService, groupsService)
	backendv1connect.RegisterParkServiceHandler(server, parkHandler)

	issuesRepo := repos.NewIssuesRepo(db)
	issueService := services.NewIssueService(issuesRepo)
	issuesHandler := handlers.NewIssuesHandler(issueService)
	backendv1connect.RegisterIssueServiceHandler(server, issuesHandler)

	groupMembershipQueue, err := msgq.NewDeviceGroupsQueueProducer(broker)
	if err != nil {
		log.Fatalf("failed to init group_membership queue in broker: %v", err)
	}

	devicesRepo := repos.NewDevicesRepo(db)
	devicesService := services.NewDeviceService(devicesRepo, groupMembershipQueue)
	devicesRegHandler := handlers.NewDeviceRegHandler(devicesService)
	devicesHandler := handlers.NewDevicesHandler(devicesService)
	backendv1connect.RegisterDeviceServiceHandler(server, devicesHandler)

	buildsRepo := repos.NewBuildsRepo(db)
	buildsService := services.NewBuildService(buildsRepo)
	buildsHandler := handlers.NewBuildsHandler(buildsService)
	backendv1connect.RegisterBuildServiceHandler(server, buildsHandler)

	releaseRepo := repos.NewReleaseRepo(db)
	releaseService := services.NewReleaseService(releaseRepo)
	releaseHandler := handlers.NewReleaseHandler(releaseService)
	backendv1connect.RegisterReleaseServiceHandler(server, releaseHandler)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /.well-known/jwks.json", jwksHandler.GetJWKS)
	connecthttp.Mount(mux, server)
	mux.HandleFunc("POST /devices/reg", devicesRegHandler.RegDevice)

	log.Printf("Connect server listening on :%s", env.MAIN_PORT)
	log.Fatalf("server failed: %v", http.ListenAndServe(":"+env.MAIN_PORT, mux))
}
