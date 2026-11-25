package postgres

import (
	"os"
	"strings"
	"testing"
)

// El listado de Pedidos solo usa orders_operational_list_idx si su ORDER BY repite la
// expresión del índice; un cambio de un lado sin el otro vuelve al recorrido completo.
func TestListOrderMatchesIndex(t *testing.T) {
	raw, err := os.ReadFile("../../../../../migrations/000146_orders_operational_list_index.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	expression := strings.TrimSuffix(listOrderSQL, ", created_at DESC")
	if !strings.Contains(string(raw), "("+expression+")") {
		t.Fatalf("orders_operational_list_idx no repite el orden del listado: %s", expression)
	}
}
