package postgres

import (
	"strings"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

// El filtro «listos para cobrar» del listado y la validación del cobro deben
// decidir lo mismo: si un estado entra en uno, entra en el otro.
func TestPOSPaymentReadySQLMatchesDomainRule(t *testing.T) {
	sql := posPaymentReadySQL
	for _, required := range []string{"o.completed_at IS NULL", "o.channel='mostrador'", "o.channel='salon' AND o.bill_closed_at IS NOT NULL"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("falta %q en la regla SQL: %s", required, sql)
		}
	}
	for _, status := range []string{"nuevo", "confirmado", "en_preparacion", "listo", "en_camino", "entregado"} {
		inSQL := strings.Contains(sql, "'"+status+"'")
		if ready := domain.PaymentReady("delivery", status, false, false); ready != inSQL {
			t.Fatalf("estado %q: PaymentReady=%v pero la regla SQL lo incluye=%v", status, ready, inSQL)
		}
	}
	if !domain.PaymentReady("mostrador", "nuevo", false, false) || !domain.PaymentReady("salon", "nuevo", false, true) || domain.PaymentReady("salon", "entregado", false, false) {
		t.Fatal("la regla de dominio cambió: actualiza posPaymentReadySQL")
	}
}
