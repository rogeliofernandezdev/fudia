package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type paymentMethodView struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Active          bool   `json:"active"`
	SalesEnabled    bool   `json:"salesEnabled"`
	ExpensesEnabled bool   `json:"expensesEnabled"`
	AffectsCash     bool   `json:"affectsCash"`
	SortOrder       int    `json:"sortOrder"`
}

type paymentMethodQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type paymentMethodExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func seedOrganizationPaymentMethods(ctx context.Context, q paymentMethodExecer, organizationID string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO payment_methods(
			organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		)
		SELECT $1,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_method_templates
		ON CONFLICT (organization_id,code) DO NOTHING
	`, organizationID)
	return err
}

func paymentMethodUsageColumn(usage string) (string, bool) {
	switch strings.TrimSpace(usage) {
	case "sales":
		return "sales_enabled", true
	case "expenses":
		return "expenses_enabled", true
	default:
		return "", false
	}
}

func getPaymentMethod(ctx context.Context, q paymentMethodQueryRower, organizationID, code, usage string) (paymentMethodView, error) {
	column, ok := paymentMethodUsageColumn(usage)
	if !ok {
		return paymentMethodView{}, pgx.ErrNoRows
	}
	var item paymentMethodView
	err := q.QueryRow(ctx, `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE organization_id=$1 AND code=$2 AND active AND `+column+`
	`, organizationID, strings.TrimSpace(code)).Scan(
		&item.Code, &item.Name, &item.Description, &item.Active,
		&item.SalesEnabled, &item.ExpensesEnabled, &item.AffectsCash, &item.SortOrder,
	)
	return item, err
}

func (a *API) paymentMethodOptions(ctx context.Context, organizationID, usage string) ([]map[string]string, error) {
	column, ok := paymentMethodUsageColumn(usage)
	if !ok {
		return []map[string]string{}, nil
	}
	rows, err := a.db.Query(ctx, `
		SELECT code,name FROM payment_methods
		WHERE organization_id=$1 AND active AND `+column+`
		ORDER BY sort_order,name
	`, organizationID)
	if err != nil { return nil, err }
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil { return nil, err }
		items = append(items, map[string]string{"value": code, "label": name})
	}
	return items, rows.Err()
}

func (a *API) listPaymentMethods(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	column, ok := paymentMethodUsageColumn(r.URL.Query().Get("usage"))
	if !ok {
		fail(w, 400, "invalid_payment_method_usage", "Indica si necesitas medios para ventas o gastos.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE organization_id=$1 AND active AND `+column+`
		ORDER BY sort_order,name
	`, s.OrganizationID)
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
		return
	}
	defer rows.Close()
	items := []paymentMethodView{}
	for rows.Next() {
		var item paymentMethodView
		if err := rows.Scan(&item.Code,&item.Name,&item.Description,&item.Active,&item.SalesEnabled,&item.ExpensesEnabled,&item.AffectsCash,&item.SortOrder); err != nil {
			fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
