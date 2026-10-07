package httpapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDashboardOpenAPIContractMatchesResponse(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	contractPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "api", "openapi.yaml")
	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}
	contract := string(content)
	required := "required: [salesNet, paidOrders, averageTicket, openOrders, criticalStock, purchasesToApprove, reservationsToday, kitchenPending, businessDate, operations, hourlySales, topProducts]"
	if !strings.Contains(contract, required) {
		t.Fatal("Dashboard OpenAPI schema does not require the complete handler response")
	}
	for _, schema := range []string{"DashboardHourlySale:", "DashboardTopProduct:", "DashboardOperations:"} {
		if !strings.Contains(contract, schema) {
			t.Fatalf("Dashboard OpenAPI schema is missing %s", schema)
		}
	}
	for _, field := range []string{"pendingBalance", "unpaidOrders", "partialOrders", "tablesTotal", "tablesOccupied", "kitchenConfirmed", "kitchenPreparing", "readyOrders", "deliveryPending", "deliveryInTransit", "activeCashRegisters", "openCashShifts", "cashBalance", "soldOutProducts"} {
		if !strings.Contains(contract, field+":") {
			t.Fatalf("missing operational field %s", field)
		}
	}
	if strings.Contains(contract, "required: [products, criticalStock, purchasesToApprove, currency]") {
		t.Fatal("legacy Dashboard schema is still present")
	}
}
