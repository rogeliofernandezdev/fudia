package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	deliveryapp "github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	paydetectapp "github.com/foods-platform/foods-backend/internal/modules/paydetect/application"
	paydetectpg "github.com/foods-platform/foods-backend/internal/modules/paydetect/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/config"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

var version = "dev"

const requestTimeout = 30 * time.Second
const responseTimeout = 35 * time.Second

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := config.LoadDotEnv(".env"); err != nil {
		logger.Error("configuration_unavailable", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx)
	if err != nil {
		logger.Error("database_unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	images, closeImages, err := configureProductImages(ctx, db)
	if err != nil {
		logger.Error("image_configuration_unavailable", "error", err)
		os.Exit(1)
	}
	defer closeImages()
	api, maintenance := newAPI(db, images)
	mux := api.Routes()
	go sweepDeliveryDispatch(ctx, logger, maintenance)
	go sweepPayPairing(ctx, logger, paydetectapp.NewMaintenance(paydetectpg.New(db)))
	mux.HandleFunc("GET /health", health)
	server := newServer(requestLogger(logger, requestDeadline(requestTimeout, mux)))
	logger.Info("api_started", "address", server.Addr, "version", version)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api_stopped", "error", err)
		os.Exit(1)
	}
}

// deliverySweepInterval es cada cuánto se revisan ofertas vencidas a centrales,
// pedidos listos para recoger y despachos de pedidos cancelados. Concierge
// también avanza la cascada cada vez que consulta su bandeja de salida.
const deliverySweepInterval = 20 * time.Second

type deliverySweeper interface{ Sweep(context.Context) error }

// operatorSessionPurgeInterval: las sesiones vencidas de los operadores de las
// centrales se borran como mucho una vez por hora, dentro del mismo barrido.
const operatorSessionPurgeInterval = time.Hour

// deliveryMaintenance agrupa el barrido del despacho y la limpieza de sesiones
// de las centrales. Solo la usa la goroutine del barrido (sin concurrencia).
type deliveryMaintenance struct {
	dispatcher *deliveryapp.Dispatcher
	operators  *deliveryapp.Operators
	lastPurge  time.Time
}

func (m *deliveryMaintenance) Sweep(ctx context.Context) error {
	err := m.dispatcher.Sweep(ctx)
	if time.Since(m.lastPurge) < operatorSessionPurgeInterval {
		return err
	}
	if _, purgeErr := m.operators.PurgeSessions(ctx); purgeErr != nil {
		return errors.Join(err, purgeErr)
	}
	m.lastPurge = time.Now()
	return err
}

func sweepDeliveryDispatch(ctx context.Context, logger *slog.Logger, dispatcher deliverySweeper) {
	ticker := time.NewTicker(deliverySweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dispatcher.Sweep(ctx); err != nil && ctx.Err() == nil {
				logger.Warn("delivery_sweep_failed", "error", err)
			}
		}
	}
}

// payPairingSweepInterval: los códigos de vinculación vencidos de Fudia Pay se
// purgan como mucho una vez por hora; la validez se comprueba en cada canje.
const payPairingSweepInterval = time.Hour

type payPairingSweeper interface {
	Sweep(context.Context, time.Time) (int64, error)
}

func sweepPayPairing(ctx context.Context, logger *slog.Logger, maintenance payPairingSweeper) {
	ticker := time.NewTicker(payPairingSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if removed, err := maintenance.Sweep(ctx, time.Now()); err != nil && ctx.Err() == nil {
				logger.Warn("pay_pairing_sweep_failed", "error", err)
			} else if removed > 0 {
				logger.Info("pay_pairing_sweep_removed", "removed", removed)
			}
		}
	}
}

func newServer(handler http.Handler) *http.Server {
	return &http.Server{Addr: envOrDefault("HTTP_ADDR", ":8080"), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: responseTimeout, IdleTimeout: 60 * time.Second}
}

// Cancel database work before the socket's write deadline, leaving time to send
// the handler's explicit error response instead of dropping the connection.
func requestDeadline(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok", Service: "foods-backend", Version: version})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	slowAfter := 500 * time.Millisecond
	if raw := os.Getenv("SLOW_REQUEST_MS"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			slowAfter = time.Duration(ms) * time.Millisecond
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := httpapi.NewCorrelationID()
		w.Header().Set("X-Request-ID", requestID)
		rw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(started)
		attrs := []any{
			"request_id", requestID,
			"correlation_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", duration.Milliseconds(),
		}
		if duration >= slowAfter {
			logger.Warn("http_request_slow", attrs...)
			return
		}
		logger.Info("http_request", attrs...)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
