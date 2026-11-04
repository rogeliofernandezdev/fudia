package httpapi

import (
	"context"
	"net/http/httptest"
	"testing"
)

// sendBotOrder pulsa «Enviar comanda»: los pedidos de Fudia Concierge quedan
// guardados hasta que el personal los envía a Cocina.
func sendBotOrder(t *testing.T, api *integrationAPI, owner scope, orderID string) {
	t.Helper()
	r := httptest.NewRequest("POST", "/v1/admin/orders/"+orderID+"/send", nil)
	r.SetPathValue("id", orderID)
	r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, owner))
	w := httptest.NewRecorder()
	api.handlers.Orders.SendOrder(w, r)
	if w.Code != 200 {
		t.Fatalf("send order %s: %d %s", orderID, w.Code, w.Body.String())
	}
}
