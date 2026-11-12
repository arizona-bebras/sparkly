package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/joho/godotenv"

	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/handlers"
	"sparkly/analytics/internal/msgq"
	"sparkly/analytics/internal/repos"
	"sparkly/analytics/internal/services"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	conn, err := db.Open(ctx)
	if err != nil {
		cancel()
		log.Fatalf("db init failed: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		cancel()
		log.Fatalf("db migration failed: %v", err)
	}
	cancel()

	rabbitHost := os.Getenv("RABBITMQ_HOST")
	if rabbitHost == "" {
		rabbitHost = "localhost"
	}
	rabbitPort := os.Getenv("RABBITMQ_PORT")
	if rabbitPort == "" {
		rabbitPort = "5672"
	}

	broker, err := msgq.NewMessageBroker(rabbitHost, rabbitPort)
	if err != nil {
		log.Fatalf("failed to init rabbitmq broker: %v", err)
	}
	defer broker.Close()

	groupMembershipRepo := repos.NewGroupMembershipRepo(conn)
	groupMembershipService := services.NewGroupMembershipService(groupMembershipRepo)
	deviceGroupsConsumer, err := msgq.NewDeviceGroupsQueueConsumer(broker, groupMembershipService)
	if err != nil {
		log.Fatalf("failed to init group_membership consumer: %v", err)
	}

	consumerCtx, consumerCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer consumerCancel()
	if err := deviceGroupsConsumer.Listen(consumerCtx); err != nil {
		log.Fatalf("failed to start group_membership consumer: %v", err)
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           newMux(repos.NewRawBatchesRepo(conn)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("analytics listening on :%s", port)
	log.Fatalf("server failed: %v", httpServer.ListenAndServe())
}

func newMux(store handlers.RawBatchStore) *http.ServeMux {
	server := connect.NewServer()
	analyticsv1connect.RegisterAnalyticsServiceHandler(server, analyticsv1connect.UnimplementedAnalyticsServiceHandler{})

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	mux.Handle("POST /ingest", handlers.NewIngestHandler(store))
	return mux
}
