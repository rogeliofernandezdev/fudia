package httpapi

import (
	"context"
	"errors"
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
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

// El tope diario de una opción de menú se valida por lotes (todas las opciones
// del pedido bloqueadas en una consulta y su uso leído en otra): cuenta lo
// pedido hoy en otros pedidos, no cuenta dos veces lo que el pedido editado ya
// tenía y rechaza con el nombre de la opción.
func TestComboOptionDailyQuotaAcrossOrdersAndEdits(t *testing.T) {
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
	var lomo, pollo string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,'LOMO-Q','Lomo saltado',20) RETURNING id`, s.OrganizationID).Scan(&lomo); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,'POLLO-Q','Pollo a la plancha',18) RETURNING id`, s.OrganizationID).Scan(&pollo); err != nil {
		t.Fatal(err)
	}
	lomoQuota, polloQuota := 3, 5
	created, err := catalog.SaveCombo(ctx, s, "", menudomain.ComboInput{Name: "Menú ejecutivo", Price: "15", Groups: []menudomain.ComboGroupInput{
		{Name: "Segundo", Required: true, MinSelections: 1, MaxSelections: 1, Options: []menudomain.ComboOptionInput{{ProductID: lomo, Quota: &lomoQuota}, {ProductID: pollo, Quota: &polloQuota}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := catalog.OrderCombo(ctx, s, created["id"])
	if err != nil {
		t.Fatal(err)
	}
	line := func(option string, qty float64) orderdomain.OrderItemInput {
		return orderdomain.OrderItemInput{ProductID: detail.ID, Qty: qty, Selections: []orderdomain.OrderItemSelectionInput{{GroupID: detail.Groups[0].ID, ProductID: option}}}
	}
	quotaExceeded := func(err error, option string) {
		t.Helper()
		var failure *apperror.Error
		if !errors.As(err, &failure) || failure.Code != "combo_option_quota_exceeded" || failure.Message != option+" ya alcanzó el tope de hoy en este menú." {
			t.Fatalf("expected quota exceeded for %s, got %v", option, err)
		}
	}

	first, err := orders.Create(ctx, s, orderdomain.OrderInput{Channel: "recojo", CustomerName: "Primero", Items: []orderdomain.OrderItemInput{line(lomo, 2), line(pollo, 1)}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = orders.Create(ctx, s, orderdomain.OrderInput{Channel: "recojo", CustomerName: "Segundo", Items: []orderdomain.OrderItemInput{line(pollo, 4), line(lomo, 2)}})
	quotaExceeded(err, "Lomo saltado")
	if _, err = orders.Create(ctx, s, orderdomain.OrderInput{Channel: "recojo", CustomerName: "Segundo", Items: []orderdomain.OrderItemInput{line(pollo, 4), line(lomo, 1)}}); err != nil {
		t.Fatalf("the remaining quota of every option must be sellable: %v", err)
	}
	// Editar el primer pedido: conserva sus dos lomos aunque el tope ya se alcanzó,
	// pero no puede sumar uno más ni otro pollo.
	update := func(items ...orderdomain.OrderItemInput) error {
		_, updateErr := orders.Update(ctx, s, first.ID, orderdomain.OrderUpdateInput{CustomerName: "Primero", Items: items})
		return updateErr
	}
	if err = update(line(lomo, 2), line(pollo, 1)); err != nil {
		t.Fatalf("an edited order keeps what it already had: %v", err)
	}
	quotaExceeded(update(line(lomo, 3), line(pollo, 1)), "Lomo saltado")
	quotaExceeded(update(line(lomo, 2), line(pollo, 2)), "Pollo a la plancha")
	if err = update(line(lomo, 1)); err != nil {
		t.Fatalf("lowering the quantity is always allowed: %v", err)
	}
}
