package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCashTimestampsDoNotDependOnDatabaseSessionTimezone(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var registerID, shiftID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Timestamp test',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	const openedAt = "2026-10-03T02:15:30.123456Z"
	const closedAt = "2026-10-03T04:45:15.654321Z"
	if err := pool.QueryRow(ctx, `INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,business_date,opening_amount,opened_by,opened_at)
		VALUES($1,$2,$3,'2026-10-02',0,$4,$5::timestamptz) RETURNING id`, s.OrganizationID, s.LocationID, registerID, s.UserID, openedAt).Scan(&shiftID); err != nil {
		t.Fatal(err)
	}
	for _, timezone := range []string{"UTC", "America/Lima", "Asia/Kathmandu"} {
		t.Run(timezone, func(t *testing.T) {
			config := pool.Config().Copy()
			config.ConnConfig.RuntimeParams["timezone"] = timezone
			zonedPool, err := pgxpool.NewWithConfig(ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			defer zonedPool.Close()
			api := New(zonedPool)
			req := httptest.NewRequest("GET", "/v1/admin/cash-shifts/"+shiftID, nil)
			req.SetPathValue("id", shiftID)
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
			get := func() cashShiftView {
				t.Helper()
				rec := httptest.NewRecorder()
				api.getCashShift(rec, req)
				if rec.Code != 200 {
					t.Fatalf("detail status=%d body=%s", rec.Code, rec.Body.String())
				}
				var shift cashShiftView
				if err := json.Unmarshal(rec.Body.Bytes(), &shift); err != nil {
					t.Fatal(err)
				}
				return shift
			}
			if _, err := pool.Exec(ctx, `UPDATE cash_shifts SET status='open',closed_at=NULL,closed_by=NULL,
				closing_expected_amount=NULL,closing_counted_amount=NULL,variance_amount=NULL WHERE id=$1`, shiftID); err != nil {
				t.Fatal(err)
			}
			open := get()
			if open.OpenedAt != openedAt || open.ClosedAt != nil || open.BusinessDate != "2026-10-02" {
				t.Fatalf("opening instant or business date changed with DB timezone: %#v", open)
			}
			if _, err := pool.Exec(ctx, `UPDATE cash_shifts SET status='closed',closed_at=$2::timestamptz,closed_by=$3,
				closing_expected_amount=0,closing_counted_amount=0,variance_amount=0 WHERE id=$1`, shiftID, closedAt, s.UserID); err != nil {
				t.Fatal(err)
			}
			closed := get()
			if closed.OpenedAt != openedAt || closed.ClosedAt == nil || *closed.ClosedAt != closedAt {
				t.Fatalf("opening/closing timestamps changed with DB timezone: %#v", closed)
			}
			for _, value := range []string{closed.OpenedAt, *closed.ClosedAt} {
				if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
					t.Fatalf("invalid RFC3339 timestamp: %q, err=%v", value, err)
				}
			}
		})
	}
}
