package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/validate"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"

	"sparkly/backend/internal/db"
	"sparkly/backend/internal/domain"
	"sparkly/backend/internal/handlers"
	"sparkly/backend/internal/infrastructure/stepca"
	"sparkly/backend/internal/interceptors"
	"sparkly/backend/internal/msgq"
	"sparkly/backend/internal/repos"
	"sparkly/backend/internal/services"
	"sparkly/backend/proto/v1/backendv1connect"
)

func main() {
	godotenv.Load()

	db, err := db.NewPostgresConn(db.DSN{
		Host:     os.Getenv("POSTGRES_HOST"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		Port:     os.Getenv("POSTGRES_PORT"),
		SSLMode:  "disable",
		TimeZone: "Asia/Yekaterinburg",
	})
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}

	broker, err := msgq.NewMessageBroker(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("failed to init rabbitmq broker: %v", err)
	}
	defer broker.Close()

	rustfsCfg := aws.Config{
		Region: os.Getenv("RUSTFS_REGION"),
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			os.Getenv("RUSTFS_ACCESS_KEY"),
			os.Getenv("RUSTFS_SECRET_KEY"),
			"",
		)),
	}

	rustfsClient := s3.NewFromConfig(rustfsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("RUSTFS_ENDPOINT_URL"))
		o.UsePathStyle = true
	})

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = rustfsClient.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(domain.ELF_OBJECTS_BUCKET_NAME),
	})

	if err != nil {
		var owned *s3types.BucketAlreadyOwnedByYou
		var exists *s3types.BucketAlreadyExists
		if errors.As(err, &owned) {
			log.Printf("you already own bucket %s.\n", domain.ELF_OBJECTS_BUCKET_NAME)
			err = owned
		} else if errors.As(err, &exists) {
			log.Printf("bucket %s already exists.\n", domain.ELF_OBJECTS_BUCKET_NAME)
		} else {
			log.Fatalf("failed to create %s rustfs bucket: %v", domain.ELF_OBJECTS_BUCKET_NAME, err)
		}
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
	deviceCA, err := stepca.New(stepca.Config{
		URL:         os.Getenv("STEP_CA_URL"),
		RootFile:    os.Getenv("STEP_CA_ROOT_FILE"),
		Provisioner: os.Getenv("STEP_CA_PROVISIONER"),
		KeyID:       os.Getenv("STEP_CA_KEY_ID"),
		KeyIDFile:   os.Getenv("STEP_CA_KEY_ID_FILE"),
		KeyFile:     os.Getenv("STEP_CA_KEY_FILE"),
	})
	if err != nil {
		log.Fatalf("failed to initialize device CA: %v", err)
	}
	devicesService := services.NewDeviceService(devicesRepo, deviceCA, groupMembershipQueue)
	devicesRegHandler := handlers.NewDeviceRegHandler(devicesService)
	devicesHandler := handlers.NewDevicesHandler(devicesService)
	backendv1connect.RegisterDeviceServiceHandler(server, devicesHandler)

	buildsRepo := repos.NewBuildsRepo(db, rustfsClient)
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

	mainPort := os.Getenv("MAIN_PORT")
	if mainPort == "" {
		mainPort = "8080"
	}
	log.Printf("Connect server listening on :%s", mainPort)
	log.Fatalf("server failed: %v", http.ListenAndServe(":"+mainPort, mux))
}
