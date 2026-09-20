package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Printf("connect error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	var orgID, locID string
	_ = conn.QueryRow(ctx, `SELECT id FROM organizations LIMIT 1`).Scan(&orgID)
	_ = conn.QueryRow(ctx, `SELECT id FROM locations WHERE organization_id=$1 LIMIT 1`, orgID).Scan(&locID)
	fmt.Printf("Org ID: %s, Loc ID: %s\n", orgID, locID)

	fmt.Printf("Ready\n")
}
