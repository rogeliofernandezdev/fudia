package main

import (
	"context"
	"fmt"
	"os"

	"github.com/foods-platform/foods-backend/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config.LoadDotEnv(".env")
	db, _ := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	defer db.Close()
	rows, _ := db.Query(context.Background(), `SELECT u.email,u.full_name,u.platform_admin,u.active,o.trade_name FROM users u JOIN organizations o ON o.id=u.organization_id ORDER BY u.created_at`)
	for rows.Next() {
		var e, n, o string
		var p, a bool
		rows.Scan(&e, &n, &p, &a, &o)
		fmt.Printf("%-32s %-24s platform=%v active=%v org=%s\n", e, n, p, a, o)
	}
	rows.Close()
}
