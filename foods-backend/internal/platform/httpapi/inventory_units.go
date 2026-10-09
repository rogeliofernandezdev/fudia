package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type inventoryUnitView struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
type inventoryUnitInput struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var inventoryUnitCode = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

func normalizeInventoryUnit(in inventoryUnitInput) (inventoryUnitInput, string) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if !inventoryUnitCode.MatchString(in.Code) {
		return in, "El código debe tener hasta 32 caracteres: letras, números, guion o guion bajo."
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 80 {
		return in, "Ingresa un nombre de hasta 80 caracteres."
	}
	return in, ""
}

func (a *API) listInventoryUnits(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `SELECT id,code,name FROM inventory_units WHERE organization_id=$1 ORDER BY sort_order,name,code`, s.OrganizationID)
	if err != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos cargar las unidades.")
		return
	}
	defer rows.Close()
	items := []inventoryUnitView{}
	for rows.Next() {
		var item inventoryUnitView
		if err = rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			fail(w, 503, "inventory_units_unavailable", "No pudimos cargar las unidades.")
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos cargar las unidades.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) createInventoryUnit(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in inventoryUnitInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_inventory_unit", "Revisa los datos de la unidad.")
		return
	}
	in, invalid := normalizeInventoryUnit(in)
	if invalid != "" {
		fail(w, 400, "invalid_inventory_unit", invalid)
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos guardar la unidad.")
		return
	}
	defer tx.Rollback(r.Context())
	var out inventoryUnitView
	err = tx.QueryRow(r.Context(), `INSERT INTO inventory_units(organization_id,code,name,sort_order) VALUES($1,$2,$3,1000) RETURNING id,code,name`, s.OrganizationID, in.Code, in.Name).Scan(&out.ID, &out.Code, &out.Name)
	if err != nil {
		var dbErr *pgconn.PgError
		if errors.As(err, &dbErr) && dbErr.Code == "23505" {
			fail(w, 409, "inventory_unit_conflict", "Ya existe una unidad con ese código.")
		} else {
			fail(w, 503, "inventory_units_unavailable", "No pudimos guardar la unidad.")
		}
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type)
	 SELECT organization_id,$2,code FROM inventory_presentation_types WHERE organization_id=$1 AND code IN ('package','box')`, s.OrganizationID, out.Code); err != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos preparar las presentaciones de la unidad.")
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'inventory_unit.created','inventory_unit',$4,jsonb_build_object('code',$5::text,'name',$6::text))`, s.OrganizationID, s.LocationID, s.UserID, out.ID, out.Code, out.Name); err != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos guardar la unidad.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "inventory_units_unavailable", "No pudimos guardar la unidad.")
		return
	}
	writeJSON(w, 201, out)
}

func validateInventoryUnit(ctx context.Context, tx pgx.Tx, organizationID, code string) *inventoryCatalogCreateError {
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_units WHERE organization_id=$1 AND code=$2)`, organizationID, code).Scan(&valid); err != nil {
		return &inventoryCatalogCreateError{Status: 503, Code: "inventory_units_unavailable", Message: "No pudimos validar la unidad base."}
	}
	if !valid {
		return &inventoryCatalogCreateError{Status: 400, Code: "invalid_inventory_unit", Message: "Selecciona una unidad base registrada en tu empresa."}
	}
	return nil
}
