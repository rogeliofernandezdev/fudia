package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
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

type paymentMethodInput struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
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

var paymentMethodCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		items = append(items, map[string]string{"value": code, "label": name})
	}
	return items, rows.Err()
}

func (a *API) listOperationalPaymentMethods(w http.ResponseWriter, r *http.Request) {
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
		if err := rows.Scan(
			&item.Code, &item.Name, &item.Description, &item.Active,
			&item.SalesEnabled, &item.ExpensesEnabled, &item.AffectsCash, &item.SortOrder,
		); err != nil {
			fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) listPaymentMethodsAdmin(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "active" && status != "inactive" {
		fail(w, 400, "invalid_payment_method_filter", "El estado del medio de pago no es válido.")
		return
	}
	where := `organization_id=$1
		AND ($2='' OR code ILIKE '%'||$2||'%' OR name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
		AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM payment_methods WHERE `+where, s.OrganizationID, q, status).Scan(&total); err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_methods
		WHERE `+where+`
		ORDER BY active DESC,sort_order,name
		LIMIT $4 OFFSET $5
	`, s.OrganizationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
		return
	}
	defer rows.Close()
	items := []paymentMethodView{}
	for rows.Next() {
		var item paymentMethodView
		if err := rows.Scan(
			&item.Code, &item.Name, &item.Description, &item.Active,
			&item.SalesEnabled, &item.ExpensesEnabled, &item.AffectsCash, &item.SortOrder,
		); err != nil {
			fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func normalizePaymentMethodInput(in *paymentMethodInput) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
}

func validPaymentMethodInput(in paymentMethodInput, creating bool) bool {
	if creating && !paymentMethodCodePattern.MatchString(in.Code) {
		return false
	}
	if in.Name == "" || len(in.Name) > 80 || len(in.Description) > 180 || in.SortOrder < 0 || in.SortOrder > 9999 {
		return false
	}
	return in.SalesEnabled || in.ExpensesEnabled
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

func (a *API) createPaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.savePaymentMethod(w, r, false)
}

func (a *API) updatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	a.savePaymentMethod(w, r, true)
}

func (a *API) savePaymentMethod(w http.ResponseWriter, r *http.Request, updating bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in paymentMethodInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_payment_method", "Revisa los datos del medio de pago.")
		return
	}
	normalizePaymentMethodInput(&in)
	code := in.Code
	if updating {
		code = strings.TrimSpace(r.PathValue("code"))
		if !paymentMethodCodePattern.MatchString(code) || (in.Code != "" && in.Code != code) {
			fail(w, 400, "immutable_payment_method_code", "El código interno no puede modificarse.")
			return
		}
	}
	if !validPaymentMethodInput(in, !updating) {
		fail(w, 400, "invalid_payment_method", "Completa un código, nombre, uso y orden válidos.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos guardar el medio de pago.")
		return
	}
	defer tx.Rollback(r.Context())

	var item paymentMethodView
	if updating {
		item, err = scanPaymentMethod(tx.QueryRow(r.Context(), `
			UPDATE payment_methods
			SET name=$3,description=$4,sales_enabled=$5,expenses_enabled=$6,affects_cash=$7,sort_order=$8,updated_at=now()
			WHERE organization_id=$1 AND code=$2
			RETURNING code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		`, s.OrganizationID, code, in.Name, in.Description, in.SalesEnabled, in.ExpensesEnabled, in.AffectsCash, in.SortOrder))
	} else {
		item, err = scanPaymentMethod(tx.QueryRow(r.Context(), `
			INSERT INTO payment_methods(
				organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
			) VALUES($1,$2,$3,$4,true,$5,$6,$7,$8)
			RETURNING code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		`, s.OrganizationID, code, in.Name, in.Description, in.SalesEnabled, in.ExpensesEnabled, in.AffectsCash, in.SortOrder))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "payment_method_not_found", "El medio de pago no existe.")
		return
	}
	if err != nil {
		fail(w, 409, "payment_method_conflict", "Ya existe un medio de pago con ese código.")
		return
	}
	ok, err := paymentMethodContinuity(r.Context(), tx, s.OrganizationID)
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos validar la configuración de medios de pago.")
		return
	}
	if !ok {
		fail(w, 409, "payment_method_required", "Debe quedar al menos un medio activo para ventas y uno para gastos.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos guardar el medio de pago.")
		return
	}
	writeJSON(w, map[bool]int{true: 200, false: 201}[updating], item)
}

func (a *API) updatePaymentMethodStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_payment_method_status", "El estado no es válido.")
		return
	}
	code := strings.TrimSpace(r.PathValue("code"))
	if !paymentMethodCodePattern.MatchString(code) {
		fail(w, 404, "payment_method_not_found", "El medio de pago no existe.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `
		UPDATE payment_methods SET active=$3,updated_at=now()
		WHERE organization_id=$1 AND code=$2
	`, s.OrganizationID, code, in.Active)
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "payment_method_not_found", "El medio de pago no existe.")
		return
	}
	ok, err := paymentMethodContinuity(r.Context(), tx, s.OrganizationID)
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos validar la configuración de medios de pago.")
		return
	}
	if !ok {
		fail(w, 409, "payment_method_required", "Debe quedar al menos un medio activo para ventas y uno para gastos.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
