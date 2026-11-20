package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// poolStatsInterval: cada cuánto se revisa si las peticiones esperaron por una
// conexión libre del pool.
const poolStatsInterval = time.Minute

type poolStater interface{ Stat() *pgxpool.Stat }

// poolWaits resume las esperas por conexión ocurridas entre dos lecturas.
type poolWaits struct {
	waits    int64
	waited   time.Duration // tiempo total de adquisición (incluye las que no esperaron)
	acquired int64
}

func poolDelta(previous, current *pgxpool.Stat) poolWaits {
	return poolWaits{
		waits:    current.EmptyAcquireCount() - previous.EmptyAcquireCount(),
		waited:   current.AcquireDuration() - previous.AcquireDuration(),
		acquired: current.AcquireCount() - previous.AcquireCount(),
	}
}

// watchPoolSaturation registra db_pool_saturated cuando, en el último minuto,
// alguna petición tuvo que esperar porque todas las conexiones estaban ocupadas.
// Es la medición que debe preceder a subir DATABASE_MAX_CONNS: sin esperas, más
// conexiones no aceleran nada (y la base remota acepta como máximo 40).
func watchPoolSaturation(ctx context.Context, logger *slog.Logger, pool poolStater) {
	ticker := time.NewTicker(poolStatsInterval)
	defer ticker.Stop()
	previous := pool.Stat()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := pool.Stat()
			delta := poolDelta(previous, current)
			previous = current
			if delta.waits <= 0 {
				continue
			}
			logger.Warn("db_pool_saturated",
				"waits", delta.waits,
				"acquires", delta.acquired,
				"acquire_ms_total", delta.waited.Milliseconds(),
				"max_conns", current.MaxConns(),
				"acquired_conns", current.AcquiredConns(),
			)
		}
	}
}
