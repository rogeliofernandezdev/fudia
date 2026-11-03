package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// La carta de la comanda muestra la descripción y los alérgenos del plato sin una consulta adicional.
func TestAvailabilityIncludesDescriptionAndAllergens(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	detailed := seedPortionAvailability(t, pool, s, 5, 0)
	plain := seedPortionAvailability(t, pool, s, 5, 0)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE products SET description='Pollo deshilachado en crema de ají amarillo.',allergens=ARRAY['Leche','Gluten'] WHERE organization_id=$1 AND id=$2`, s.OrganizationID, detailed); err != nil {
		t.Fatalf("seed product details: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET description='',allergens=NULL WHERE organization_id=$1 AND id=$2`, s.OrganizationID, plain); err != nil {
		t.Fatalf("seed product without details: %v", err)
	}
	request := httptest.NewRequest("GET", "/v1/admin/product-availability?pageSize=100&productIds="+detailed+","+plain, nil)
	request = request.WithContext(context.WithValue(request.Context(), scopeKey{}, s))
	response := httptest.NewRecorder()
	New(pool).listProductAvailability(response, request)
	if response.Code != 200 {
		t.Fatalf("availability: %d %s", response.Code, response.Body.String())
	}
	var out struct {
		Items []struct {
			ProductID   string    `json:"productId"`
			Description *string   `json:"description"`
			Allergens   *[]string `json:"allergens"`
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	byID := map[string]int{}
	for index, item := range out.Items {
		byID[item.ProductID] = index
	}
	first, ok := byID[detailed]
	if !ok {
		t.Fatalf("detailed product missing: %s", response.Body.String())
	}
	item := out.Items[first]
	if item.Description == nil || *item.Description != "Pollo deshilachado en crema de ají amarillo." || item.Allergens == nil || len(*item.Allergens) != 2 || (*item.Allergens)[0] != "Leche" || (*item.Allergens)[1] != "Gluten" {
		t.Fatalf("details not returned: %s", response.Body.String())
	}
	second, ok := byID[plain]
	if !ok {
		t.Fatalf("plain product missing: %s", response.Body.String())
	}
	item = out.Items[second]
	if item.Description == nil || *item.Description != "" || item.Allergens == nil || len(*item.Allergens) != 0 {
		t.Fatalf("missing details must be empty, not null: %s", response.Body.String())
	}
}
