package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/joho/godotenv"

	"sparkly/analytics/internal/handlers"
	"sparkly/analytics/proto/v1/analyticsv1connect"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           newMux(os.Stdout),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("analytics listening on :%s", port)
	log.Fatalf("server failed: %v", httpServer.ListenAndServe())
}

func newMux(output io.Writer) *http.ServeMux {
	server := connect.NewServer()
	analyticsv1connect.RegisterAnalyticsServiceHandler(server, analyticsv1connect.UnimplementedAnalyticsServiceHandler{})

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	mux.Handle("POST /ingest", handlers.NewIngestHandler(output))
	return mux
}
