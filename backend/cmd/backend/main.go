package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"sparkly/backend/internal/db"
)

func main() {
	env := struct {
		POSTGRES_DB       string
		POSTGRES_USER     string
		POSTGRES_PASSWORD string
		POSTGRES_PORT     string
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
	}

	readEnv()

	_, err := db.NewPostgresConn(db.DSN{
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
}
