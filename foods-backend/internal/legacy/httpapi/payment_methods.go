package httpapi

import (
	"context"
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"strings"
)

type paymentMethodView = billingdomain.PaymentMethodView
type paymentMethodInput = billingdomain.PaymentMethodInput

var paymentMethodCodePattern = billingdomain.PaymentMethodCodePattern
var normalizePaymentMethodInput = billingdomain.NormalizePaymentMethodInput
var validPaymentMethodInput = billingdomain.ValidPaymentMethodInput

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
func scanPaymentMethod(row pgx.Row) (paymentMethodView, error) {
	var item paymentMethodView
	err := row.Scan(
		&item.Code, &item.Name, &item.Description, &item.Active,
		&item.SalesEnabled, &item.ExpensesEnabled, &item.AffectsCash, &item.SortOrder,
	)
	return item, err
}
func getPaymentMethod(ctx context.Context, q paymentMethodQueryRower, organizationID, code, usage string) (paymentMethodView, error) {
	column, ok := paymentMethodUsageColumn(usage)
	if !ok {
		return paymentMethodView{}, pgx.ErrNoRows
	}
	return scanPaymentMethod(q.QueryRow(ctx, `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE organization_id=$1 AND code=$2 AND active AND `+column+`
	`, organizationID, strings.TrimSpace(code)))
}
func paymentMethodContinuity(ctx context.Context, q paymentMethodQueryRower, organizationID string) (bool, error) {
	var sales, expenses int
	err := q.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE active AND sales_enabled),
		       count(*) FILTER (WHERE active AND expenses_enabled)
		FROM payment_methods
		WHERE organization_id=$1
	`, organizationID).Scan(&sales, &expenses)
	return sales > 0 && expenses > 0, err
}

type PaymentMethodsHandler interface {
	ListOperationalPaymentMethods(http.ResponseWriter, *http.Request)
	ListPaymentMethodsAdmin(http.ResponseWriter, *http.Request)
	CreatePaymentMethod(http.ResponseWriter, *http.Request)
	UpdatePaymentMethod(http.ResponseWriter, *http.Request)
	UpdatePaymentMethodStatus(http.ResponseWriter, *http.Request)
	Options(context.Context, string, string) ([]map[string]string, error)
}

func (a *API) listOperationalPaymentMethods(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.ListOperationalPaymentMethods(w, r)
}
func (a *API) listPaymentMethodsAdmin(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.ListPaymentMethodsAdmin(w, r)
}
func (a *API) createPaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.CreatePaymentMethod(w, r)
}
func (a *API) updatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.UpdatePaymentMethod(w, r)
}
func (a *API) updatePaymentMethodStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.PaymentMethods.UpdatePaymentMethodStatus(w, r)
}
func (a *API) paymentMethodOptions(ctx context.Context, org, usage string) ([]map[string]string, error) {
	return a.handlers.PaymentMethods.Options(ctx, org, usage)
}
func (a *API) savePaymentMethod(w http.ResponseWriter, r *http.Request, updating bool) {
	if updating {
		a.updatePaymentMethod(w, r)
	} else {
		a.createPaymentMethod(w, r)
	}
}
