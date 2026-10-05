package httpapi

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := pgxpool.New(ctx, databaseURL)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "test setup database:", err)
		os.Exit(1)
	}

	const phone = "+999000000001"
	_, err = pool.Exec(context.Background(), `
		INSERT INTO platform_whatsapp_channels(country_code,phone_number,phone_number_id,display_name,secret_ref,active)
		VALUES('PE',$1,'test-platform-phone-id','Integration Test Global',NULL,true)
		ON CONFLICT(phone_number) DO UPDATE SET
			country_code='PE',
			phone_number_id='test-platform-phone-id',
			secret_ref=NULL,
			active=true,
			updated_at=now()
	`, phone)
	if err != nil {
		pool.Close()
		fmt.Fprintln(os.Stderr, "test setup WhatsApp fixture:", err)
		os.Exit(1)
	}

	code := m.Run()
	_, _ = pool.Exec(context.Background(), `DELETE FROM platform_whatsapp_channels WHERE phone_number=$1`, phone)
	pool.Close()
	os.Exit(code)
}
