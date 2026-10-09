package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type expenseCategoryView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type expenseView struct {
	ID                string  `json:"id"`
	CategoryID        string  `json:"categoryId"`
	CategoryName      string  `json:"categoryName"`
	Description       string  `json:"description"`
	Amount            string  `json:"amount"`
	PaymentMethod     string  `json:"paymentMethod"`
	BusinessDate      string  `json:"businessDate"`
	Reference         string  `json:"reference"`
	Notes             string  `json:"notes"`
	Status            string  `json:"status"`
	CreatedByName     string  `json:"createdByName"`
	CreatedAt         string  `json:"createdAt"`
	VoidedByName      string  `json:"voidedByName"`
	VoidedAt          *string `json:"voidedAt"`
	VoidReason        string  `json:"voidReason"`
}

type expenseInput struct {
	CategoryID    string `json:"categoryId"`
	Description   string `json:"description"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"paymentMethod"`
	BusinessDate  string `json:"businessDate"`
	Reference     string `json:"reference"`
	Notes         string `json:"notes"`
}

var expenseAmountPattern = regexp.MustCompile("^[0-9]{1,12}([.][0-9]{1,2})?$")

const expenseColumns = `
	e.id::text,e.category_id::text,c.name,e.description,e.amount::text,e.payment_method,
	e.business_date::text,e.reference,e.notes,e.status,created.full_name,
	to_char(e.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	COALESCE(voided.full_name,''),
	CASE WHEN e.voided_at IS NULL THEN NULL ELSE to_char(e.voided_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,
	e.void_reason
`

func scanExpense(row pgx.Row) (expenseView, error) {
	var item expenseView
	err := row.Scan(
		&item.ID, &item.CategoryID, &item.CategoryName, &item.Description, &item.Amount,
		&item.PaymentMethod, &item.BusinessDate, &item.Reference, &item.Notes, &item.Status,
		&item.CreatedByName, &item.CreatedAt, &item.VoidedByName, &item.VoidedAt, &item.VoidReason,
	)
	return item, err
}

func validExpenseAmount(value string) bool {
	if !expenseAmountPattern.MatchString(value) {
		return false
	}
	plain := strings.ReplaceAll(value, ".", "")
	return strings.Trim(plain, "0") != ""
}

func (a *API) loadExpense(r *http.Request, id string) (expenseView, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	return scanExpense(a.db.QueryRow(r.Context(), `
		SELECT ` + expenseColumns + `
		FROM expenses e
		JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id
		JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id
		LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id
		WHERE e.id=$1 AND e.organization_id=$2 AND e.location_id=$3
	`, id, s.OrganizationID, s.LocationID))
}

func (a *API) listExpenses(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	categoryID := strings.TrimSpace(r.URL.Query().Get("categoryId"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if status != "" && status != "active" && status != "void" {
		fail(w, 400, "invalid_expense_filter", "El estado de gasto no es válido.")
		return
	}
	for _, value := range []string{from, to} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				fail(w, 400, "invalid_expense_filter", "Revisa el rango de fechas.")
				return
			}
		}
	}
	where := `
		e.organization_id=$1 AND e.location_id=$2
		AND ($3='' OR e.description ILIKE '%'||$3||'%' OR e.reference ILIKE '%'||$3||'%' OR c.name ILIKE '%'||$3||'%' OR created.full_name ILIKE '%'||$3||'%')
		AND ($4='' OR e.category_id::text=$4)
		AND ($5='' OR e.status=$5)
		AND ($6='' OR e.business_date>=NULLIF($6,'')::date)
		AND ($7='' OR e.business_date<=NULLIF($7,'')::date)
	`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM expenses e JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id WHERE ` + where, s.OrganizationID, s.LocationID, q, categoryID, status, from, to).Scan(&total); err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos cargar los gastos.")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT ` + expenseColumns + ` FROM expenses e JOIN expense_categories c ON c.id=e.category_id AND c.organization_id=e.organization_id JOIN users created ON created.id=e.created_by AND created.organization_id=e.organization_id LEFT JOIN users voided ON voided.id=e.voided_by AND voided.organization_id=e.organization_id WHERE ` + where + ` ORDER BY e.business_date DESC,e.created_at DESC,e.id DESC LIMIT $8 OFFSET $9`, s.OrganizationID, s.LocationID, q, categoryID, status, from, to, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos cargar los gastos.")
		return
	}
	defer rows.Close()
	items := []expenseView{}
	for rows.Next() {
		item, scanErr := scanExpense(rows)
		if scanErr != nil {
			fail(w, 503, "expenses_unavailable", "No pudimos cargar los gastos.")
			return
		}
		items = append(items, item)
	}
	options, err := a.paymentMethodOptions(r.Context(), s.OrganizationID, "expenses")
	if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
		return
	}
	writeJSON(w, 200, map[string]any{
		"items":                items,
		"total":                total,
		"page":                 page,
		"pageSize":             size,
		"paymentMethodOptions": options,
	})
}

func (a *API) createExpense(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in expenseInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_expense", "Revisa los datos del gasto.")
		return
	}
	in.CategoryID = strings.TrimSpace(in.CategoryID)
	in.Description = strings.TrimSpace(in.Description)
	in.Amount = strings.TrimSpace(in.Amount)
	in.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
	in.BusinessDate = strings.TrimSpace(in.BusinessDate)
	in.Reference = strings.TrimSpace(in.Reference)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.CategoryID == "" || in.Description == "" || len(in.Description) > 180 || !validExpenseAmount(in.Amount) || in.PaymentMethod == "" || len(in.Reference) > 120 || len(in.Notes) > 500 {
		fail(w, 400, "invalid_expense", "Completa categoría, descripción, importe y medio de pago válidos.")
		return
	}
	if _, err := time.Parse("2006-01-02", in.BusinessDate); err != nil {
		fail(w, 400, "invalid_expense", "La fecha del gasto no es válida.")
		return
	}
	if _, err := getPaymentMethod(r.Context(), a.db, s.OrganizationID, in.PaymentMethod, "expenses"); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "invalid_payment_method", "Selecciona un medio de pago activo para gastos.")
		return
	} else if err != nil {
		fail(w, 503, "payment_methods_unavailable", "No pudimos validar el medio de pago.")
		return
	}
	var categoryActive bool
	err := a.db.QueryRow(r.Context(), `SELECT active FROM expense_categories WHERE id::text=$1 AND organization_id=$2`, in.CategoryID, s.OrganizationID).Scan(&categoryActive)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "expense_category_not_found", "La categoría seleccionada no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos validar la categoría.")
		return
	}
	if !categoryActive {
		fail(w, 409, "expense_category_inactive", "Selecciona una categoría activa.")
		return
	}
	var id string
	err = a.db.QueryRow(r.Context(), `
		INSERT INTO expenses(organization_id,location_id,category_id,business_date,description,amount,payment_method,reference,notes,created_by)
		VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, in.CategoryID, in.BusinessDate, in.Description, in.Amount, in.PaymentMethod, in.Reference, in.Notes, s.UserID).Scan(&id)
	if err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos registrar el gasto.")
		return
	}
	a.audit(r, "created", "expense", id)
	item, err := a.loadExpense(r, id)
	if err != nil {
		fail(w, 503, "expenses_unavailable", "El gasto se registró, pero no pudimos cargar su detalle.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) voidExpense(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Reason string `json:"reason"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_expense_void", "Indica el motivo de anulación.")
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if len(in.Reason) < 4 || len(in.Reason) > 240 {
		fail(w, 400, "invalid_expense_void", "El motivo debe tener entre 4 y 240 caracteres.")
		return
	}
	id := r.PathValue("id")
	var status string
	err := a.db.QueryRow(r.Context(), `SELECT status FROM expenses WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "expense_not_found", "El gasto no existe en este local.")
		return
	}
	if err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos cargar el gasto.")
		return
	}
	if status == "void" {
		fail(w, 409, "expense_already_void", "El gasto ya está anulado.")
		return
	}
	if _, err = a.db.Exec(r.Context(), `UPDATE expenses SET status='void',voided_by=$4,voided_at=now(),void_reason=$5,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='active'`, id, s.OrganizationID, s.LocationID, s.UserID, in.Reason); err != nil {
		fail(w, 503, "expenses_unavailable", "No pudimos anular el gasto.")
		return
	}
	a.audit(r, "voided", "expense", id)
	item, err := a.loadExpense(r, id)
	if err != nil {
		fail(w, 503, "expenses_unavailable", "El gasto se anuló, pero no pudimos cargar su detalle.")
		return
	}
	writeJSON(w, 200, item)
}

func (a *API) listExpenseCategories(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && status != "active" && status != "inactive" {
		fail(w, 400, "invalid_expense_category_filter", "El estado de categoría no es válido.")
		return
	}
	where := `organization_id=$1 AND ($2='' OR name ILIKE '%'||$2||'%') AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM expense_categories WHERE ` + where, s.OrganizationID, q, status).Scan(&total); err != nil {
		fail(w, 503, "expense_categories_unavailable", "No pudimos cargar las categorías.")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT id::text,name,active FROM expense_categories WHERE ` + where + ` ORDER BY active DESC,name LIMIT $4 OFFSET $5`, s.OrganizationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "expense_categories_unavailable", "No pudimos cargar las categorías.")
		return
	}
	defer rows.Close()
	items := []expenseCategoryView{}
	for rows.Next() {
		var item expenseCategoryView
		if rows.Scan(&item.ID, &item.Name, &item.Active) != nil {
			fail(w, 503, "expense_categories_unavailable", "No pudimos cargar las categorías.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) createExpenseCategory(w http.ResponseWriter, r *http.Request) {
	a.saveExpenseCategory(w, r, false)
}

func (a *API) updateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	a.saveExpenseCategory(w, r, true)
}

func (a *API) saveExpenseCategory(w http.ResponseWriter, r *http.Request, updating bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_expense_category", "Revisa el nombre de la categoría.")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 100 {
		fail(w, 400, "invalid_expense_category", "El nombre debe tener entre 1 y 100 caracteres.")
		return
	}
	var item expenseCategoryView
	var err error
	if updating {
		err = a.db.QueryRow(r.Context(), `UPDATE expense_categories SET name=$3,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING id::text,name,active`, r.PathValue("id"), s.OrganizationID, in.Name).Scan(&item.ID, &item.Name, &item.Active)
	} else {
		err = a.db.QueryRow(r.Context(), `INSERT INTO expense_categories(organization_id,name) VALUES($1,$2) RETURNING id::text,name,active`, s.OrganizationID, in.Name).Scan(&item.ID, &item.Name, &item.Active)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "expense_category_not_found", "La categoría no existe.")
		return
	}
	if err != nil {
		fail(w, 409, "expense_category_conflict", "Ya existe una categoría con ese nombre.")
		return
	}
	a.audit(r, map[bool]string{true: "updated", false: "created"}[updating], "expense_category", item.ID)
	writeJSON(w, map[bool]int{true: 200, false: 201}[updating], item)
}

func (a *API) updateExpenseCategoryStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_expense_category_status", "El estado no es válido.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE expense_categories SET active=$3,updated_at=now() WHERE id=$1 AND organization_id=$2`, r.PathValue("id"), s.OrganizationID, in.Active)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "expense_category_not_found", "La categoría no existe.")
		return
	}
	a.audit(r, "status_updated", "expense_category", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}
