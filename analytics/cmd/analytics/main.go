package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/joho/godotenv"

	"sparkly/analytics/internal/config"
	"sparkly/analytics/internal/db"
	"sparkly/analytics/internal/handlers"
	"sparkly/analytics/internal/msgq"
	"sparkly/analytics/internal/parcel"
	"sparkly/analytics/internal/repos"
	"sparkly/analytics/internal/services"
	"sparkly/analytics/proto/v1/analyticsv1connect"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		log.Printf("failed to load .env file: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	conn, err := db.Open(ctx)
	if err != nil {
		cancel()
		return fmt.Errorf("db init failed: %w", err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		cancel()
		return fmt.Errorf("db migration failed: %w", err)
	}
	cancel()

	broker, err := msgq.NewMessageBroker(cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to init rabbitmq broker: %w", err)
	}
	defer broker.Close()

	ingestPublisher, err := msgq.NewIngestPublisher(broker)
	if err != nil {
		return fmt.Errorf("failed to init ingest publisher: %w", err)
	}

	ingestService := services.NewIngestService(parcel.Codec{}, ingestPublisher)
	rawPersistence := services.NewRawBatchPersistence(repos.NewRawBatchesRepo(conn))
	rawConsumer, err := msgq.NewRawBatchesWorkers(broker, rawPersistence, cfg.RawPersistenceWorkers)
	if err != nil {
		return fmt.Errorf("failed to init raw persistence consumer: %w", err)
	}
	defer rawConsumer.Close()

	groupMembershipRepo := repos.NewGroupMembershipRepo(conn)
	groupMembershipService := services.NewGroupMembershipService(groupMembershipRepo)
	deviceGroupsConsumer, err := msgq.NewDeviceGroupsQueueConsumer(broker, groupMembershipService)
	if err != nil {
		return fmt.Errorf("failed to init group_membership consumer: %w", err)
	}

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rawConsumer.Listen(appCtx); err != nil {
		return fmt.Errorf("failed to start raw persistence consumer: %w", err)
	}

	log.Printf("raw persistence running with %d workers", cfg.RawPersistenceWorkers)

	if err := deviceGroupsConsumer.Listen(appCtx); err != nil {
		return fmt.Errorf("failed to start group_membership consumer: %w", err)
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newMux(ingestService),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-appCtx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("analytics listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

func newMux(ingester handlers.ParcelIngester) *http.ServeMux {
	server := connect.NewServer()
	analyticsv1connect.RegisterAnalyticsServiceHandler(server, analyticsv1connect.UnimplementedAnalyticsServiceHandler{})

	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	mux.Handle("POST /ingest", handlers.NewIngestHandler(ingester))
	return mux
}
