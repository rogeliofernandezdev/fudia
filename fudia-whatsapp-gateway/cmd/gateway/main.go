package main

import (
	"context"
	"github.com/foods-platform/fudia-whatsapp-gateway/internal/gateway"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := gateway.LoadConfig()
	if err != nil {
		logger.Error("configuration incomplete")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	transport, err := gateway.Open(ctx, cfg)
	if err != nil {
		logger.Error("session storage or ownership unavailable")
		os.Exit(1)
	}
	defer transport.Close()
	server := &http.Server{Addr: cfg.Address, Handler: transport.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP listener unavailable")
			cancel()
		}
	}()
	logger.Info("WhatsApp transport started")
	<-ctx.Done()
	shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	_ = server.Shutdown(shutdown)
}
