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
	required := "required: [salesNet, paidOrders, averageTicket, openOrders, criticalStock, purchasesToApprove, reservationsToday, kitchenPending, hourlySales, topProducts]"
	if !strings.Contains(contract, required) {
		t.Fatal("Dashboard OpenAPI schema does not require the complete handler response")
	}
	for _, schema := range []string{"DashboardHourlySale:", "DashboardTopProduct:"} {
		if !strings.Contains(contract, schema) {
			t.Fatalf("Dashboard OpenAPI schema is missing %s", schema)
		}
	}
	if strings.Contains(contract, "required: [products, criticalStock, purchasesToApprove, currency]") {
		t.Fatal("legacy Dashboard schema is still present")
	}
}
