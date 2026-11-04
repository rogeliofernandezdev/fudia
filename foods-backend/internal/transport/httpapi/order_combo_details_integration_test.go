package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	menuapp "github.com/foods-platform/foods-backend/internal/modules/menu/application"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	menupg "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

// GET /order-combo-details devuelve en una petición lo mismo que GET
// /order-combos/{id} para cada menú, en el orden pedido y con los grupos y
// opciones en su orden, y falla todo el lote si un menú no se puede vender.
func TestOrderComboDetailsMatchSingleCombos(t *testing.T) {
	ctx := context.Background()
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combo_groups WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combos WHERE organization_id=$1`, s.OrganizationID)
	})
	catalog := menuapp.NewCatalog(menupg.NewCatalog(pool), database.NewUnitOfWork(pool), auditpg.New(pool), nil)
	product := func(sku, name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,$2,$3,10) RETURNING id`, s.OrganizationID, sku, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	soup, salad, rice, chicken := product("DET-SOPA", "Sopa"), product("DET-ENSALADA", "Ensalada"), product("DET-ARROZ", "Arroz"), product("DET-POLLO", "Pollo")
	quota := 5
	save := func(name string, groups []menudomain.ComboGroupInput) string {
		t.Helper()
		created, err := catalog.SaveCombo(ctx, s, "", menudomain.ComboInput{Name: name, Price: "15", Groups: groups})
		if err != nil {
			t.Fatal(err)
		}
		return created["id"]
	}
	lunch := save("Menú ejecutivo", []menudomain.ComboGroupInput{
		{Name: "Entrada", Required: true, MinSelections: 1, MaxSelections: 1, Options: []menudomain.ComboOptionInput{{ProductID: soup}, {ProductID: salad, Quota: &quota}}},
		{Name: "Segundo", Required: true, MinSelections: 1, MaxSelections: 1, Options: []menudomain.ComboOptionInput{{ProductID: chicken}, {ProductID: rice}}},
	})
	kids := save("Menú niños", []menudomain.ComboGroupInput{
		{Name: "Plato", Required: true, MinSelections: 1, MaxSelections: 1, Options: []menudomain.ComboOptionInput{{ProductID: rice}}},
	})

	call := func(query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/admin/order-combo-details?"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		New(pool).getOrderComboDetails(rec, req)
		return rec
	}
	rec := call("ids=" + kids + "," + lunch)
	if rec.Code != 200 {
		t.Fatalf("details: %d %s", rec.Code, rec.Body.String())
	}
	var batch struct {
		Items []menudomain.OrderCombo `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &batch); err != nil {
		t.Fatal(err)
	}
	if len(batch.Items) != 2 || batch.Items[0].ID != kids || batch.Items[1].ID != lunch {
		t.Fatalf("details must follow the requested order: %+v", batch.Items)
	}
	for i, id := range []string{kids, lunch} {
		single, err := catalog.OrderCombo(ctx, s, id)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(single)
		var want menudomain.OrderCombo
		_ = json.Unmarshal(raw, &want)
		if !reflect.DeepEqual(batch.Items[i], want) {
			t.Fatalf("batch detail differs from the single detail:\n%+v\n%+v", batch.Items[i], want)
		}
	}
	// Grupos y opciones conservan su orden aunque ahora se lean en una consulta.
	got := []string{}
	for _, group := range batch.Items[1].Groups {
		for _, option := range group.Options {
			got = append(got, group.Name+":"+option.Name)
		}
	}
	if strings.Join(got, ",") != "Entrada:Sopa,Entrada:Ensalada,Segundo:Pollo,Segundo:Arroz" {
		t.Fatalf("groups/options out of order: %v", got)
	}
	if option := batch.Items[1].Groups[0].Options[1]; option.QuotaRemaining == nil || *option.QuotaRemaining != 5 {
		t.Fatalf("daily quota must be reported: %+v", option)
	}

	for _, query := range []string{"", "ids=", "ids=invalid", "ids=" + kids + "," + kids, "ids=" + strings.Repeat(kids+",", 20) + lunch} {
		if rec := call(query); rec.Code != 400 {
			t.Fatalf("%q: expected 400, got %d %s", query, rec.Code, rec.Body.String())
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET active=false WHERE id=$1`, kids); err != nil {
		t.Fatal(err)
	}
	if rec := call(fmt.Sprintf("ids=%s,%s", lunch, kids)); rec.Code != 409 {
		t.Fatalf("an unavailable menu must fail the batch like the single detail: %d %s", rec.Code, rec.Body.String())
	}
}
