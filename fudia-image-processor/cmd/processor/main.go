package main

import (
	"context"
	"github.com/foods-platform/fudia-image-processor/internal/application"
	"github.com/foods-platform/fudia-image-processor/internal/infrastructure/gcs"
	"github.com/foods-platform/fudia-image-processor/internal/infrastructure/imaging"
	"github.com/foods-platform/fudia-image-processor/internal/infrastructure/postgres"
	"github.com/foods-platform/fudia-image-processor/internal/transport/events"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	connection := os.Getenv("DATABASE_URL")
	if connection == "" {
		slog.Error("DATABASE_URL required")
		os.Exit(1)
	}
	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		slog.Error("invalid database configuration")
		os.Exit(1)
	}
	config.MaxConns = 2
	config.MinConns = 1
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("database_unavailable")
		os.Exit(1)
	}
	defer db.Close()
	if err = db.Ping(ctx); err != nil {
		slog.Error("database_unavailable")
		os.Exit(1)
	}
	bucket := os.Getenv("PRODUCT_IMAGES_BUCKET")
	store, err := gcs.New(ctx, bucket)
	if err != nil {
		slog.Error("storage_unavailable", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	processor := application.NewImageProcessor(postgres.NewImages(db), store, imaging.Vips{})
	mux := http.NewServeMux()
	mux.Handle("POST /", events.Images(processor, bucket))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 130 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("processor_stopped", "error", err)
		os.Exit(1)
	}
}
