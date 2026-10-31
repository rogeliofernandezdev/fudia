package httpapi

import (
	"context"
	"testing"

	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	inventoryapp "github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	inventorypg "github.com/foods-platform/foods-backend/internal/modules/inventory/infrastructure/postgres"
	menuapp "github.com/foods-platform/foods-backend/internal/modules/menu/application"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	menupg "github.com/foods-platform/foods-backend/internal/modules/menu/infrastructure/postgres"
	orderapp "github.com/foods-platform/foods-backend/internal/modules/orders/application"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	orderpg "github.com/foods-platform/foods-backend/internal/modules/orders/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

func TestComboStockSharedPortionsAndExhaustedOptions(t *testing.T) {
	ctx := context.Background()
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combo_groups WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM menu_combos WHERE organization_id=$1`, s.OrganizationID)
	})
	work, audit := database.NewUnitOfWork(pool), auditpg.New(pool)
	catalog := menuapp.NewCatalog(menupg.NewCatalog(pool), work, audit, nil)
	orders := orderapp.New(orderpg.New(pool), work, audit, inventoryapp.NewStock(inventorypg.NewStock(pool), work, audit))
	var dish string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price,quantity_control) VALUES($1,'AJI-TEST','Ají de gallina',10,'portions') RETURNING id`, s.OrganizationID).Scan(&dish); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,sold_quantity) SELECT $1,l.id,$3,(now() AT TIME ZONE l.timezone)::date,3,0 FROM locations l WHERE l.id=$2`, s.OrganizationID, s.LocationID, dish); err != nil {
		t.Fatal(err)
	}
	quota := 3
	definition := menudomain.ComboInput{Name: "Menú prueba", Price: "12", Groups: []menudomain.ComboGroupInput{{Name: "Segundo", Required: true, MinSelections: 1, MaxSelections: 1, Options: []menudomain.ComboOptionInput{{ProductID: dish, Quota: &quota}}}}}
	created, err := catalog.SaveCombo(ctx, s, "", definition)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := catalog.OrderCombo(ctx, s, created["id"])
	if err != nil {
		t.Fatal(err)
	}
	option := detail.Groups[0].Options[0]
	if !option.Available || option.Remaining == nil || *option.Remaining != 3 || option.QuotaRemaining == nil || *option.QuotaRemaining != 3 {
		t.Fatalf("expected three physical portions and three menu portions, got %+v", option)
	}
	comboLine := orderdomain.OrderItemInput{ProductID: detail.ID, Qty: 8, Selections: []orderdomain.OrderItemSelectionInput{{GroupID: detail.Groups[0].ID, ProductID: dish}}}
	create := func(items ...orderdomain.OrderItemInput) error {
		_, createErr := orders.Create(ctx, s, orderdomain.OrderInput{Channel: "recojo", CustomerName: "Cliente prueba", Items: items})
		return createErr
	}
	if err := create(comboLine); err == nil {
		t.Fatal("eight combos must exceed the three-option quota")
	}
	comboLine.Qty = 3
	if err := create(comboLine, orderdomain.OrderItemInput{ProductID: dish, Qty: 1}); err == nil {
		t.Fatal("three combos plus a dish must exceed shared physical stock")
	}
	var sold float64
	if err := pool.QueryRow(ctx, `SELECT sold_quantity FROM product_availability WHERE product_id=$1 AND location_id=$2`, dish, s.LocationID).Scan(&sold); err != nil || sold != 0 {
		t.Fatalf("rejected orders must roll back stock: sold=%v err=%v", sold, err)
	}
	comboLine.Qty = 2
	if err := create(comboLine, orderdomain.OrderItemInput{ProductID: dish, Qty: 1}); err != nil {
		t.Fatal(err)
	}
	detail, err = catalog.OrderCombo(ctx, s, detail.ID)
	if err != nil {
		t.Fatal(err)
	}
	option = detail.Groups[0].Options[0]
	if option.Available || option.Remaining == nil || *option.Remaining != 0 || option.QuotaRemaining == nil || *option.QuotaRemaining != 1 {
		t.Fatalf("physical exhaustion must disable option even with remaining menu quota: %+v", option)
	}
	definition.Name = "Otro menú con plato agotado"
	if _, err := catalog.SaveCombo(ctx, s, "", definition); err == nil {
		t.Fatal("cannot register a new menu with an exhausted option")
	}
	comboLine.Qty = 1
	if err := create(comboLine); err == nil {
		t.Fatal("cannot sell an exhausted option")
	}
}

func TestValidateComboRejectsZeroQuota(t *testing.T) {
	zero := 0
	in := comboInput{Name: "Menú", Price: "10", Groups: []comboGroupInput{{Name: "Segundo", Required: true, MinSelections: 1, MaxSelections: 1, Options: []comboOptionInput{{ProductID: "dish", Quota: &zero}}}}}
	if validateCombo(in) == "" {
		t.Fatal("a zero reservation must not be accepted")
	}
}
