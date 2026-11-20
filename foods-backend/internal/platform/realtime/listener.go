package realtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Listen mantiene una conexión dedicada con LISTEN foods_live y publica cada
// aviso en el hub. Si la conexión se cae, marca el hub como no disponible
// (las pantallas vuelven al polling) y reintenta con espera creciente hasta que
// ctx termine. La conexión se saca del pool (Hijack): queda ocupada esperando
// avisos y no debe volver a usarse para consultas.
//
// Requiere una conexión directa a PostgreSQL: un PgBouncer en modo transacción
// acepta LISTEN pero no entrega los avisos. En ese caso las pantallas siguen
// actualizándose con su consulta de respaldo.
func Listen(ctx context.Context, pool *pgxpool.Pool, hub *Hub, logger *slog.Logger) {
	wait := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := listenOnce(ctx, pool, hub)
		hub.SetReady(false)
		if ctx.Err() != nil {
			return
		}
		if time.Since(started) > time.Minute {
			wait = time.Second
		}
		logger.Warn("live_listener_lost", "error", err, "retry_in_ms", wait.Milliseconds())
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if wait < 30*time.Second {
			wait *= 2
		}
	}
}

func listenOnce(ctx context.Context, pool *pgxpool.Pool, hub *Hub) error {
	pooled, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	conn := pooled.Hijack()
	defer conn.Close(context.Background())
	if _, err = conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	hub.SetReady(true)
	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if change, ok := ParseChange(notification.Payload); ok {
			hub.Publish(change)
		}
	}
}
