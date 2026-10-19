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
	"sparkly/backend/internal/metadata"
	"sparkly/backend/internal/repos"
	"sparkly/backend/internal/services"
	"sparkly/backend/proto/v1/backendv1connect"
)

func main() {
	env := struct {
		POSTGRES_DB       string
		POSTGRES_USER     string
		POSTGRES_PASSWORD string
		POSTGRES_PORT     string
		PORT              string
	}{}

	readEnv := func() {
		err := godotenv.Load()
		if err != nil {
			log.Printf("failed to load .env file: %v", err)
		}
		env.POSTGRES_DB, _ = os.LookupEnv("POSTGRES_DB")
		env.POSTGRES_USER, _ = os.LookupEnv("POSTGRES_USER")
		env.POSTGRES_PASSWORD, _ = os.LookupEnv("POSTGRES_PASSWORD")
		env.POSTGRES_PORT, _ = os.LookupEnv("POSTGRES_PORT")
		env.PORT, _ = os.LookupEnv("PORT")
		if env.PORT == "" {
			env.PORT = "8080"
		}
	}
	readEnv()

	_, ok := os.LookupEnv("JWT_SECRET")
	if !ok {
		log.Fatalf("jwt secret not found")
	}

	db, err := db.NewPostgresConn(db.DSN{
		Host:     "localhost",
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

	server := connect.NewServer(
		validate.NewServerInterceptor(),
		metadata.NewAuthInterceptor(),
	)

	userRepo := repos.NewUserRepo(db)
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)
	backendv1connect.RegisterUserServiceHandler(server, userHandler)

	parkRepo := repos.NewParkRepo(db)
	parkService := services.NewParkService(parkRepo)
	parkHandler := handlers.NewParkHandler(parkService)
	backendv1connect.RegisterParkServiceHandler(server, parkHandler)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	log.Printf("Connect server listening on :%s", env.PORT)
	log.Fatalf("server failed: %v", http.ListenAndServe(":"+env.PORT, mux))
}
