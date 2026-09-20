package main

import (
	"context"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/platform/config"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		exit("load configuration", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		exit("connect database", err)
	}
	defer db.Close()
	if _, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		exit("prepare migration history", err)
	}
	files, err := filepath.Glob(filepath.Join("migrations", "*.up.sql"))
	if err != nil {
		exit("find migrations", err)
	}
	sort.Strings(files)
	for _, path := range files {
		version := strings.TrimSuffix(filepath.Base(path), ".up.sql")
		var applied bool
		if err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&applied); err != nil {
			exit("read migration history", err)
		}
		if applied {
			fmt.Printf("migration %s already applied\n", version)
			continue
		}
		sql, readErr := os.ReadFile(path)
		if readErr != nil {
			exit("read migration", readErr)
		}
		tx, beginErr := db.Begin(ctx)
		if beginErr != nil {
			exit("begin migration", beginErr)
		}
		if _, err = tx.Exec(ctx, string(sql)); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, version)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			exit("apply migration "+version, err)
		}
		if err = tx.Commit(ctx); err != nil {
			exit("commit migration "+version, err)
		}
		fmt.Printf("migration %s applied\n", version)
	}
	fmt.Println("database is up to date")
}
func exit(step string, err error) { fmt.Fprintf(os.Stderr, "%s: %v\n", step, err); os.Exit(1) }
