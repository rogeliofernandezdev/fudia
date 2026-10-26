package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedProductCategory crea una categoría activa: los productos ya no aceptan
// «Sin categoría» al crearse ni al editarse.
func seedProductCategory(t *testing.T, pool *pgxpool.Pool, organizationID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO menu_categories(organization_id,name,product_scope) VALUES($1,$2,'both') RETURNING id`, organizationID, fmt.Sprintf("Categoría %s", organizationID[:8])).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// withProductCategory agrega categoryId a un cuerpo JSON de producto.
func withProductCategory(body, categoryID string) string {
	return strings.Replace(body, "{", fmt.Sprintf(`{"categoryId":%q,`, categoryID), 1)
}

func postProduct(api *integrationAPI, s scope, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/v1/admin/products", bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createProduct(rec, req)
	return rec
}

func TestCreateProductRejectsInvalidInitialPortionsBeforeDatabase(t *testing.T) {
	for _, quantity := range []string{"", `,"initialPortionQuantity":null`, `,"initialPortionQuantity":0`, `,"initialPortionQuantity":-1`, `,"initialPortionQuantity":1.5`, `,"initialPortionQuantity":"15"`, `,"initialPortionQuantity":2147483648`} {
		if rec := postProduct(New(nil), scope{}, `{"name":"Ají de gallina","price":"20","quantityControl":"portions"`+quantity+`}`); rec.Code != 400 {
			t.Fatalf("invalid quantity %s: %d %s", quantity, rec.Code, rec.Body.String())
		}
	}
	for _, control := range []string{"none", "inventory"} {
		if rec := postProduct(New(nil), scope{}, `{"name":"Producto","price":"20","quantityControl":"`+control+`","initialPortionQuantity":15}`); rec.Code != 400 {
			t.Fatalf("unsupported quantity control %s: %d", control, rec.Code)
		}
	}
}
