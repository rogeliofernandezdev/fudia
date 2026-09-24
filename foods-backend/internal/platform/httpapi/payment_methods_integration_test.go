package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPaymentMethodsAreDatabaseDriven(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO payment_methods(
			organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		) VALUES($1,'voucher','Vale interno','Medio creado desde datos',true,true,true,false,55)
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	method, err := getPaymentMethod(ctx, pool, s.OrganizationID, "voucher", "sales")
	if err != nil {
		t.Fatalf("database-defined payment method should be accepted: %v", err)
	}
	if method.Code != "voucher" || method.Name != "Vale interno" || method.AffectsCash {
		t.Fatalf("unexpected method: %#v", method)
	}
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET active=false WHERE organization_id=$1 AND code='voucher'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := getPaymentMethod(ctx, pool, s.OrganizationID, "voucher", "sales"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("inactive database method must be rejected, got %v", err)
	}
}
