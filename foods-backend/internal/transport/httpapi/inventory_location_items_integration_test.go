package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// locationItemsHarness calls the inventory handlers as an authenticated user
// of one location, the way the admin web does after choosing a location.
type locationItemsHarness struct {
	t    *testing.T
	api  *integrationAPI
	pool *pgxpool.Pool
	a, b scope
}

func newLocationItemsHarness(t *testing.T) *locationItemsHarness {
	t.Helper()
	pool := integrationPool(t)
	a := seedInventoryScope(t, pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	var second string
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Sucursal Norte',$2,'') RETURNING id`,
		a.OrganizationID, fmt.Sprintf("N%d", nonce%1000000)).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE locations SET name='Central' WHERE id=$1`, a.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, statement := range []string{
			`DELETE FROM product_recipe_items WHERE organization_id=$1`,
			`DELETE FROM product_recipes WHERE organization_id=$1`,
			`DELETE FROM inventory_transfer_items WHERE organization_id=$1`,
			`DELETE FROM inventory_transfers WHERE organization_id=$1`,
			`DELETE FROM inventory_location_settings WHERE organization_id=$1`,
		} {
			_, _ = pool.Exec(context.Background(), statement, a.OrganizationID)
		}
	})
	b := a
	b.LocationID = second
	return &locationItemsHarness{t: t, api: New(pool), pool: pool, a: a, b: b}
}

func (h *locationItemsHarness) call(actor scope, method, path, body string, handler http.HandlerFunc, values map[string]string) *httptest.ResponseRecorder {
	h.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range values {
		req.SetPathValue(key, value)
	}
	req = req.WithContext(context.WithValue(context.Background(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func (h *locationItemsHarness) create(actor scope, body string) inventoryProductOption {
	h.t.Helper()
	rec := h.call(actor, "POST", "/v1/admin/purchase-inventory-items", body, h.api.createPurchaseInventoryItem, nil)
	var item inventoryProductOption
	if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &item) != nil {
		h.t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	return item
}

func (h *locationItemsHarness) detail(actor scope, id string) (int, inventorydomain.ItemDetail) {
	h.t.Helper()
	rec := h.call(actor, "GET", "/v1/admin/inventory/items/"+id, "", h.api.getInventoryItem, map[string]string{"id": id})
	var out inventorydomain.ItemDetail
	if rec.Code == 200 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			h.t.Fatal(err)
		}
	}
	return rec.Code, out
}

func (h *locationItemsHarness) listed(actor scope, status, id string) bool {
	h.t.Helper()
	rec := h.call(actor, "GET", "/v1/admin/inventory?pageSize=100&status="+status, "", h.api.listInventory, nil)
	if rec.Code != 200 {
		h.t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	return strings.Contains(rec.Body.String(), id)
}

func (h *locationItemsHarness) entry(actor scope, id, unit string, quantity float64) {
	h.t.Helper()
	body := fmt.Sprintf(`{"inventoryItemId":%q,"unit":%q,"quantity":%v,"unitCost":2}`, id, unit, quantity)
	if rec := h.call(actor, "POST", "/v1/admin/inventory/entries", body, h.api.createInventoryEntry, nil); rec.Code != 201 {
		h.t.Fatalf("entry: %d %s", rec.Code, rec.Body.String())
	}
}

func (h *locationItemsHarness) exit(actor scope, id string, quantity float64) *httptest.ResponseRecorder {
	h.t.Helper()
	body := fmt.Sprintf(`{"inventoryItemId":%q,"movementType":"exit","reason":"waste","quantity":%v}`, id, quantity)
	return h.call(actor, "POST", "/v1/admin/inventory/adjustments", body, h.api.createInventoryAdjustment, nil)
}

func (h *locationItemsHarness) status(actor scope, id string, active bool) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.call(actor, "PATCH", "/v1/admin/inventory/items/"+id+"/status", fmt.Sprintf(`{"active":%v}`, active), h.api.updateInventoryItemStatus, map[string]string{"id": id})
}

func (h *locationItemsHarness) settings(actor scope, id, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.call(actor, "PATCH", "/v1/admin/inventory/items/"+id+"/settings", body, h.api.updateInventoryLocationSettings, map[string]string{"id": id})
}

func (h *locationItemsHarness) preview(actor scope, id, unit string) inventorydomain.UnitChangePreview {
	h.t.Helper()
	rec := h.call(actor, "GET", "/v1/admin/inventory/items/"+id+"/unit-change?unit="+unit, "", h.api.previewInventoryUnitChange, map[string]string{"id": id})
	var out inventorydomain.UnitChangePreview
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
		h.t.Fatalf("preview: %d %s", rec.Code, rec.Body.String())
	}
	return out
}

func (h *locationItemsHarness) grantManage(actor scope) {
	h.t.Helper()
	_, err := h.pool.Exec(context.Background(), `WITH role AS (
	 INSERT INTO roles(organization_id,name,permissions,menu_access)
	 VALUES($1,$4,ARRAY['inventory.read','inventory.manage'],ARRAY['inventario']) RETURNING id
	) INSERT INTO user_roles(user_id,role_id,location_id) SELECT $2,id,$3 FROM role`,
		actor.OrganizationID, actor.UserID, actor.LocationID, fmt.Sprintf("Almacén %d", time.Now().UnixNano()))
	if err != nil {
		h.t.Fatal(err)
	}
}

func TestInventoryArticlesBelongToTheLocationThatUsesThem(t *testing.T) {
	h := newLocationItemsHarness(t)
	lemon := h.create(h.a, `{"newIngredient":{"name":"Limón"},"unit":"kg"}`)

	if !h.listed(h.a, "active", lemon.ID) || h.listed(h.b, "all", lemon.ID) {
		t.Fatal("a new article must appear only in the location that created it")
	}
	if code, _ := h.detail(h.b, lemon.ID); code != 404 {
		t.Fatalf("other location must not open the article: %d", code)
	}
	picker := h.call(h.b, "GET", "/v1/admin/purchase-inventory-items?q=Lim", "", h.api.listPurchaseInventoryItems, nil)
	if picker.Code != 200 || strings.Contains(picker.Body.String(), lemon.ID) {
		t.Fatalf("purchase picker of another location lists the article: %d %s", picker.Code, picker.Body.String())
	}
	if lookup := h.call(h.b, "GET", "/v1/admin/inventory/products?q=Lim&scope=location", "", h.api.listInventoryProducts, nil); strings.Contains(lookup.Body.String(), lemon.ID) {
		t.Fatalf("adjustment lookup of another location lists the article: %s", lookup.Body.String())
	}
	// Recipes are company-wide: their lookup keeps every company article.
	if recipes := h.call(h.b, "GET", "/v1/admin/inventory/products?q=Lim", "", h.api.listInventoryProducts, nil); !strings.Contains(recipes.Body.String(), lemon.ID) {
		t.Fatalf("recipe lookup must list company articles: %s", recipes.Body.String())
	}
	found := h.call(h.b, "GET", "/v1/admin/purchase-assignable-items?q=lim", "", h.api.listAssignableItems, nil)
	if found.Code != 200 || !strings.Contains(found.Body.String(), lemon.ID) || !strings.Contains(found.Body.String(), "Central") {
		t.Fatalf("assignable search: %d %s", found.Code, found.Body.String())
	}

	// Without other locations and without history, it can be deleted.
	if _, d := h.detail(h.a, lemon.ID); !d.CanDelete || !d.CanDeactivate || d.HasHistory {
		t.Fatalf("fresh article rules: %+v", d)
	}
	if rec := h.call(h.b, "POST", "/v1/admin/purchase-inventory-items/"+lemon.ID+"/assignment", "", h.api.assignInventoryItem, map[string]string{"id": lemon.ID}); rec.Code != 204 {
		t.Fatalf("assign: %d %s", rec.Code, rec.Body.String())
	}
	if !h.listed(h.b, "active", lemon.ID) {
		t.Fatal("assigned article must appear in the second location")
	}
	if _, d := h.detail(h.a, lemon.ID); d.CanDelete || !strings.Contains(d.DeleteBlocker, "Sucursal Norte") || len(d.OtherLocations) != 1 {
		t.Fatalf("shared article must not be deletable: %+v", d)
	}
	if rec := h.call(h.a, "DELETE", "/v1/admin/inventory/items/"+lemon.ID, "", h.api.deleteInventoryItem, map[string]string{"id": lemon.ID}); rec.Code != 409 {
		t.Fatalf("delete shared: %d %s", rec.Code, rec.Body.String())
	}

	// A company-wide rename needs permission in the other location.
	if rec := h.settings(h.a, lemon.ID, `{"name":"Limón sutil","minimumStock":1,"optimalStock":2}`); rec.Code != 403 || !strings.Contains(rec.Body.String(), "Sucursal Norte") {
		t.Fatalf("rename without permission in other location: %d %s", rec.Code, rec.Body.String())
	}
	h.grantManage(h.b)
	if rec := h.settings(h.a, lemon.ID, `{"name":"Limón sutil","minimumStock":1,"optimalStock":2}`); rec.Code != 204 {
		t.Fatalf("rename with permission: %d %s", rec.Code, rec.Body.String())
	}
	// Stock levels stay per location.
	if rec := h.settings(h.b, lemon.ID, `{"minimumStock":5,"optimalStock":9}`); rec.Code != 204 {
		t.Fatalf("levels in b: %d %s", rec.Code, rec.Body.String())
	}
	if _, d := h.detail(h.a, lemon.ID); d.MinimumStock != "1" || d.Name != "Limón sutil" {
		t.Fatalf("levels of a changed by b: %+v", d)
	}

	// Deactivation is per location and requires zero stock.
	h.entry(h.a, lemon.ID, "kg", 3)
	if rec := h.status(h.a, lemon.ID, false); rec.Code != 409 || !strings.Contains(rec.Body.String(), "Quedan 3") {
		t.Fatalf("deactivate with stock: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.exit(h.a, lemon.ID, 3); rec.Code != 201 {
		t.Fatalf("exit: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.status(h.a, lemon.ID, false); rec.Code != 204 {
		t.Fatalf("deactivate: %d %s", rec.Code, rec.Body.String())
	}
	if h.listed(h.a, "active", lemon.ID) || !h.listed(h.a, "inactive", lemon.ID) || !h.listed(h.b, "active", lemon.ID) {
		t.Fatal("deactivation must hide the article only in its location")
	}
	if rec := h.exit(h.a, lemon.ID, 1); rec.Code != 404 {
		t.Fatalf("adjusting a deactivated article: %d %s", rec.Code, rec.Body.String())
	}
	var movements int
	if err := h.pool.QueryRow(context.Background(), `SELECT count(*) FROM stock_movements WHERE inventory_item_id=$1 AND location_id=$2`, lemon.ID, h.a.LocationID).Scan(&movements); err != nil || movements != 2 {
		t.Fatalf("kardex must be kept: %d %v", movements, err)
	}
	if _, d := h.detail(h.a, lemon.ID); d.CanDelete || !d.HasHistory {
		t.Fatalf("article with history must not be deletable: %+v", d)
	}
	if rec := h.status(h.a, lemon.ID, true); rec.Code != 204 || !h.listed(h.a, "active", lemon.ID) {
		t.Fatalf("reactivate: %d %s", rec.Code, rec.Body.String())
	}
	var audits int
	if err := h.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE entity_id=$1 AND action IN ('inventory.item_assigned','inventory.item_deactivated','inventory.item_reactivated')`, lemon.ID).Scan(&audits); err != nil || audits != 3 {
		t.Fatalf("status changes must be audited: %d %v", audits, err)
	}
}

func TestInventoryTransferAssignsTheArticleToTheDestination(t *testing.T) {
	h := newLocationItemsHarness(t)
	flour := h.create(h.a, `{"newIngredient":{"name":"Harina"},"unit":"kg"}`)
	h.entry(h.a, flour.ID, "kg", 10)
	body := fmt.Sprintf(`{"idempotencyKey":"trf-%d","toLocationId":%q,"items":[{"inventoryItemId":%q,"quantity":4}]}`, time.Now().UnixNano(), h.b.LocationID, flour.ID)
	if rec := h.call(h.a, "POST", "/v1/admin/inventory/transfers", body, h.api.createInventoryTransfer, nil); rec.Code != 201 {
		t.Fatalf("transfer: %d %s", rec.Code, rec.Body.String())
	}
	if !h.listed(h.b, "active", flour.ID) {
		t.Fatal("received transfer must add the article to the destination")
	}
	if rec := h.exit(h.b, flour.ID, 4); rec.Code != 201 {
		t.Fatalf("exit in destination: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.status(h.b, flour.ID, false); rec.Code != 204 {
		t.Fatalf("deactivate in destination: %d %s", rec.Code, rec.Body.String())
	}
	// A new receipt in the location brings the article back to its list.
	body = fmt.Sprintf(`{"idempotencyKey":"trf2-%d","toLocationId":%q,"items":[{"inventoryItemId":%q,"quantity":1}]}`, time.Now().UnixNano(), h.b.LocationID, flour.ID)
	if rec := h.call(h.a, "POST", "/v1/admin/inventory/transfers", body, h.api.createInventoryTransfer, nil); rec.Code != 201 {
		t.Fatalf("second transfer: %d %s", rec.Code, rec.Body.String())
	}
	if !h.listed(h.b, "active", flour.ID) {
		t.Fatal("stock received again must reactivate the article in the destination")
	}
}

func TestInventoryUnitCorrectionKeepsNumbersAndRespectsRecipes(t *testing.T) {
	h := newLocationItemsHarness(t)
	ctx := context.Background()
	soda := h.create(h.a, `{"newIngredient":{"name":"Gaseosa 500 ml"},"unit":"kg"}`)

	if p := h.preview(h.a, soda.ID, "botella"); p.Mode != "free" || !p.Allowed {
		t.Fatalf("free change without history: %+v", p)
	}
	h.entry(h.a, soda.ID, "kg", 2.5)
	if p := h.preview(h.a, soda.ID, "botella"); p.Mode != "correction" || p.Allowed || !strings.Contains(strings.Join(p.Blockers, " "), "decimales") {
		t.Fatalf("countable unit with fractional stock: %+v", p)
	}
	if rec := h.exit(h.a, soda.ID, 0.5); rec.Code != 201 {
		t.Fatalf("exit: %d %s", rec.Code, rec.Body.String())
	}
	if p := h.preview(h.a, soda.ID, "botella"); !p.Allowed || p.Quantity != "2" {
		t.Fatalf("correction allowed: %+v", p)
	}
	if rec := h.settings(h.a, soda.ID, `{"unit":"botella","minimumStock":1,"optimalStock":6}`); rec.Code != 204 {
		t.Fatalf("unit correction: %d %s", rec.Code, rec.Body.String())
	}
	var unit, entryUnit string
	var quantity float64
	if err := h.pool.QueryRow(ctx, `SELECT i.unit,b.quantity::float8,(SELECT DISTINCT unit FROM inventory_entries e WHERE e.inventory_item_id=i.id)
	  FROM inventory_items i JOIN stock_balances b ON b.inventory_item_id=i.id AND b.location_id=$2 WHERE i.id=$1`, soda.ID, h.a.LocationID).Scan(&unit, &quantity, &entryUnit); err != nil {
		t.Fatal(err)
	}
	if unit != "botella" || quantity != 2 || entryUnit != "botella" {
		t.Fatalf("correction must relabel without converting: unit=%s qty=%v entries=%s", unit, quantity, entryUnit)
	}
	var audited int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE entity_id=$1 AND action='inventory.unit_changed' AND metadata->>'from'='kg' AND metadata->>'to'='botella' AND metadata->>'mode'='correction'`, soda.ID).Scan(&audited); err != nil || audited != 1 {
		t.Fatalf("unit change audit: %d %v", audited, err)
	}

	// A recipe in another unit blocks the correction until it is changed.
	flour := h.create(h.a, `{"newIngredient":{"name":"Harina sin preparar"},"unit":"kg"}`)
	var productID, recipeID string
	if err := h.pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control) VALUES($1,$2,'Pan casero',3,true,'prepared','none') RETURNING id`,
		h.a.OrganizationID, fmt.Sprintf("PAN-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := h.pool.QueryRow(ctx, `INSERT INTO product_recipes(organization_id,product_id) VALUES($1,$2) RETURNING id`, h.a.OrganizationID, productID).Scan(&recipeID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO product_recipe_items(organization_id,recipe_id,inventory_item_id,quantity,consumption_unit,consumption_quantity,consumption_factor) VALUES($1,$2,$3,0.2,'g',200,0.001)`,
		h.a.OrganizationID, recipeID, flour.ID); err != nil {
		t.Fatal(err)
	}
	if p := h.preview(h.a, flour.ID, "und"); p.Allowed || !strings.Contains(strings.Join(p.Blockers, " "), "Pan casero") {
		t.Fatalf("recipe in grams must block: %+v", p)
	}
	if rec := h.settings(h.a, flour.ID, `{"unit":"und","minimumStock":0,"optimalStock":0}`); rec.Code != 409 {
		t.Fatalf("blocked unit change must fail: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.status(h.a, flour.ID, false); rec.Code != 409 || !strings.Contains(rec.Body.String(), "Pan casero") {
		t.Fatalf("recipe must block deactivation: %d %s", rec.Code, rec.Body.String())
	}
	// A recipe in the article's own unit follows the correction.
	if _, err := h.pool.Exec(ctx, `UPDATE product_recipe_items SET consumption_unit='kg',consumption_quantity=0.2,consumption_factor=1 WHERE recipe_id=$1`, recipeID); err != nil {
		t.Fatal(err)
	}
	if rec := h.settings(h.a, flour.ID, `{"unit":"g","minimumStock":0,"optimalStock":0}`); rec.Code != 204 {
		t.Fatalf("unit change with recipe in own unit: %d %s", rec.Code, rec.Body.String())
	}
	var recipeUnit string
	if err := h.pool.QueryRow(ctx, `SELECT consumption_unit FROM product_recipe_items WHERE recipe_id=$1`, recipeID).Scan(&recipeUnit); err != nil || recipeUnit != "g" {
		t.Fatalf("recipe unit must follow the correction: %q %v", recipeUnit, err)
	}
}

func TestInventoryDeleteOnlyUnusedArticles(t *testing.T) {
	h := newLocationItemsHarness(t)
	ctx := context.Background()
	draft := h.create(h.a, `{"newIngredient":{"name":"Insumo de prueba"},"unit":"botella","presentations":[{"presentationType":"unit","unitsPerPresentation":1,"isDefault":true},{"presentationType":"package","unitsPerPresentation":6}]}`)
	_, d := h.detail(h.a, draft.ID)
	var base, pack string
	for _, p := range d.Presentations {
		if p.IsBase {
			base = p.ID
		} else {
			pack = p.ID
		}
	}
	if base == "" || pack == "" {
		t.Fatalf("presentations: %+v", d.Presentations)
	}
	remove := func(id string) *httptest.ResponseRecorder {
		return h.call(h.a, "DELETE", "/v1/admin/inventory/items/"+draft.ID+"/presentations/"+id, "", h.api.removeInventoryPresentation, map[string]string{"id": draft.ID, "presentationId": id})
	}
	if rec := remove(base); rec.Code != 409 {
		t.Fatalf("base presentation removal: %d %s", rec.Code, rec.Body.String())
	}
	if rec := remove(pack); rec.Code != 200 || !strings.Contains(rec.Body.String(), "deleted") {
		t.Fatalf("unused presentation removal: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.call(h.a, "DELETE", "/v1/admin/inventory/items/"+draft.ID, "", h.api.deleteInventoryItem, map[string]string{"id": draft.ID}); rec.Code != 204 {
		t.Fatalf("delete unused: %d %s", rec.Code, rec.Body.String())
	}
	var exists bool
	if err := h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_items WHERE id=$1)`, draft.ID).Scan(&exists); err != nil || exists {
		t.Fatalf("deleted article still exists: %v %v", exists, err)
	}

	// A sold product without history: the inventory goes, the Carta product stays.
	category := seedRetailCategory(t, h.pool, h.a)
	sold := h.create(h.a, fmt.Sprintf(`{"newProduct":{"name":"Agua sin gas","categoryId":%q,"price":"2"},"unit":"botella"}`, category))
	if rec := h.call(h.a, "DELETE", "/v1/admin/inventory/items/"+sold.ID, "", h.api.deleteInventoryItem, map[string]string{"id": sold.ID}); rec.Code != 204 {
		t.Fatalf("delete linked: %d %s", rec.Code, rec.Body.String())
	}
	var control string
	if err := h.pool.QueryRow(ctx, `SELECT quantity_control FROM products WHERE id=$1`, *sold.ProductID).Scan(&control); err != nil || control != "none" {
		t.Fatalf("linked product must stay without stock control: %q %v", control, err)
	}

	used := h.create(h.a, `{"newIngredient":{"name":"Aceite"},"unit":"l"}`)
	h.entry(h.a, used.ID, "l", 1)
	if rec := h.call(h.a, "DELETE", "/v1/admin/inventory/items/"+used.ID, "", h.api.deleteInventoryItem, map[string]string{"id": used.ID}); rec.Code != 409 {
		t.Fatalf("delete with history: %d %s", rec.Code, rec.Body.String())
	}
}
