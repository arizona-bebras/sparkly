package main

import (
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/joho/godotenv"

	"sparkly/backend/internal/db"
	"sparkly/backend/internal/handlers"
	"sparkly/backend/internal/repos"
	"sparkly/backend/internal/services"
	"sparkly/backend/proto/backendv1connect"
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

	server := connect.NewServer()

	userRepo := repos.NewUserRepo(db)
	userService := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)
	backendv1connect.RegisterUserServiceHandler(server, userHandler)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)

	log.Printf("Connect server listening on :%s", env.PORT)
	if err := http.ListenAndServe(":"+env.PORT, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
