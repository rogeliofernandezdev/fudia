package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Una petición que espera porque todas las conexiones están ocupadas se cuenta
// como espera; las adquisiciones con conexión libre, no.
func TestPoolDeltaCountsOnlyAcquiresThatWaited(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurado")
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns, config.MinConns = 1, 0
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	warm, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	warm.Release()
	before := pool.Stat()
	free, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if delta := poolDelta(before, pool.Stat()); delta.waits != 0 || delta.acquired != 1 {
		t.Fatalf("a free connection is not a wait: %+v", delta)
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		free.Release()
	}()
	waited, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waited.Release()
	if delta := poolDelta(before, pool.Stat()); delta.waits != 1 || delta.acquired != 2 {
		t.Fatalf("a busy pool must record one wait: %+v", delta)
	}
}
