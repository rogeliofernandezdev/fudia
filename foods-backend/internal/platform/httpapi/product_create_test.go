package httpapi

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
)

func postProduct(api *API, s scope, body string) *httptest.ResponseRecorder {
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
