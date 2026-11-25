package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOrdersListOperationalFiltersCountsAndScope(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	seed := func(owner scope, channel, status, name, createdAt string) string {
		t.Helper()
		var id string
		err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,customer_name,total,created_by,created_at) VALUES($1,$2,$3,$4,$5,10,$6,$7::timestamptz) RETURNING id`, owner.OrganizationID, owner.LocationID, channel, status, name, owner.UserID, createdAt).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	newID := seed(s, "salon", "nuevo", "Cliente lista", "2026-01-01T01:00:00Z")
	qrID := seed(s, "whatsapp", "confirmado", "Cliente lista", "2026-01-01T02:00:00Z")
	readyID := seed(s, "mostrador", "listo", "Cliente lista", "2026-01-02T01:00:00Z")
	shippingID := seed(s, "delivery", "en_camino", "Cliente lista reparto", "2026-01-03T01:00:00Z")
	cancelledID := seed(s, "recojo", "cancelado", "Cliente lista", "2026-01-04T01:00:00Z")
	deliveredID := seed(s, "delivery", "entregado", "Cliente lista", "2026-01-05T01:00:00Z")
	servedID := seed(s, "salon", "entregado", "Cliente lista", "2026-01-06T01:00:00Z")
	seed(foreign, "delivery", "en_camino", "Cliente lista reparto", "2026-01-07T01:00:00Z")
	otherLocal := s
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,'Otro local','ORDERS-FILTER-OTHER') RETURNING id`, s.OrganizationID).Scan(&otherLocal.LocationID); err != nil {
		t.Fatal(err)
	}
	seed(otherLocal, "delivery", "en_camino", "Cliente lista reparto", "2026-01-08T01:00:00Z")
	type page struct {
		Items   []order             `json:"items"`
		Total   int                 `json:"total"`
		Counts  map[string]int      `json:"channelCounts"`
		Options []map[string]string `json:"statusOptions"`
	}
	list := func(query string) page {
		t.Helper()
		r := httptest.NewRequest("GET", "/v1/operations/orders?page=1&pageSize=20&"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.listOrders(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", query, w.Code, w.Body.String())
		}
		var result page
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	ids := func(result page) []string {
		out := []string{}
		for _, item := range result.Items {
			out = append(out, item.ID)
		}
		return out
	}
	all := list("")
	if all.Total != 7 || !reflect.DeepEqual(ids(all), []string{shippingID, readyID, qrID, newID, servedID, deliveredID, cancelledID}) {
		t.Fatalf("all includes history after pending work, independently of table payment: %+v", all)
	}
	if !reflect.DeepEqual(all.Counts, map[string]int{"salon": 3, "recojo": 2, "delivery": 2}) {
		t.Fatalf("tenant-scoped service counts: %+v", all.Counts)
	}
	for _, item := range all.Items {
		if item.OperationalStatus == "" {
			t.Fatalf("missing operational status: %+v", item)
		}
	}
	ready := list("status=listo")
	if ready.Total != 2 || !reflect.DeepEqual(ids(ready), []string{shippingID, readyID}) || !reflect.DeepEqual(ready.Counts, map[string]int{"recojo": 1, "delivery": 1}) {
		t.Fatalf("Por entregar includes shipping, counters follow the filter: %+v", ready)
	}
	if ready.Items[0].Status != "en_camino" || ready.Items[0].OperationalStatus != "listo" {
		t.Fatalf("projection must preserve delivery actions: %+v", ready.Items[0])
	}
	for _, option := range ready.Options {
		if option["value"] == "en_camino" || option["value"] == "abiertos" {
			t.Fatalf("logistics/group leaked into operational selector: %+v", ready.Options)
		}
	}
	for _, alias := range []string{"salon", "recojo", "mostrador"} {
		result := list("channel=" + alias)
		want := 2
		if alias == "salon" {
			want = 3
		}
		if result.Total != want || !reflect.DeepEqual(result.Counts, all.Counts) {
			t.Fatalf("channel %s must not restrict tab counts: %+v", alias, result)
		}
	}
	filtered := list("channel=salon&status=entregado")
	if filtered.Total != 1 || !reflect.DeepEqual(ids(filtered), []string{servedID}) || !reflect.DeepEqual(filtered.Counts, map[string]int{"salon": 1, "delivery": 1}) {
		t.Fatalf("delivered is independent of an open table account: %+v", filtered)
	}
	searched := list("q=reparto&status=listo")
	if searched.Total != 1 || !reflect.DeepEqual(ids(searched), []string{shippingID}) || !reflect.DeepEqual(searched.Counts, map[string]int{"delivery": 1}) || len(searched.Options) != 1 || searched.Options[0]["value"] != "listo" {
		t.Fatalf("search and status catalog include shipping without foreign rows: %+v", searched)
	}
	empty := list("q=reparto&status=preparando")
	if empty.Total != 0 || len(empty.Counts) != 0 || len(empty.Options) != 2 || empty.Options[0]["value"] != "preparando" {
		t.Fatalf("selected empty status stays selectable: %+v", empty)
	}
	legacy := list("status=en_camino")
	if legacy.Total != 1 || !reflect.DeepEqual(ids(legacy), []string{shippingID}) {
		t.Fatalf("legacy shipping filter remains compatible: %+v", legacy)
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, shippingID).Scan(&stored); err != nil || stored != "en_camino" {
		t.Fatalf("list must not rewrite shipment history: %s %v", stored, err)
	}
}
