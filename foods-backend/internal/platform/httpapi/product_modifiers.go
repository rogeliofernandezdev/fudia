package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type modifierOptionInput struct {
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}

type modifierGroupInput struct {
	Name          string                `json:"name"`
	Required      bool                  `json:"required"`
	MinSelections int                   `json:"minSelections"`
	MaxSelections int                   `json:"maxSelections"`
	Options       []modifierOptionInput `json:"options"`
}

type modifierConfigInput struct {
	Groups []modifierGroupInput `json:"groups"`
}

type modifierOptionView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}

type modifierGroupView struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Required      bool                 `json:"required"`
	MinSelections int                  `json:"minSelections"`
	MaxSelections int                  `json:"maxSelections"`
	Options       []modifierOptionView `json:"options"`
}

type modifierConfigView struct {
	ProductID string              `json:"productId"`
	Groups    []modifierGroupView `json:"groups"`
}

type modifierQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadProductModifiers(
	ctx context.Context,
	q modifierQuerier,
	organizationID string,
	productID string,
) (modifierConfigView, error) {
	out := modifierConfigView{ProductID: productID, Groups: []modifierGroupView{}}
	rows, err := q.Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       o.id::text,o.name,o.surcharge::text
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, organizationID, productID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	index := map[string]int{}
	for rows.Next() {
		var groupID, groupName string
		var required bool
		var minSelections, maxSelections int
		var optionID, optionName, surcharge *string
		if err := rows.Scan(
			&groupID, &groupName, &required, &minSelections, &maxSelections,
			&optionID, &optionName, &surcharge,
		); err != nil {
			return out, err
		}
		position, exists := index[groupID]
		if !exists {
			position = len(out.Groups)
			index[groupID] = position
			out.Groups = append(out.Groups, modifierGroupView{
				ID: groupID, Name: groupName, Required: required,
				MinSelections: minSelections, MaxSelections: maxSelections,
				Options: []modifierOptionView{},
			})
		}
		if optionID != nil && optionName != nil && surcharge != nil {
			out.Groups[position].Options = append(out.Groups[position].Options, modifierOptionView{
				ID: *optionID, Name: *optionName, Surcharge: *surcharge,
			})
		}
	}
	return out, rows.Err()
}

func validateModifierConfig(in modifierConfigInput) string {
	groupNames := map[string]bool{}
	for _, group := range in.Groups {
		name := strings.TrimSpace(group.Name)
		if name == "" || len([]rune(name)) > 80 {
			return "Cada grupo necesita un nombre de hasta 80 caracteres."
		}
		key := strings.ToLower(name)
		if groupNames[key] {
			return "Los grupos de modificadores necesitan nombres distintos."
		}
		groupNames[key] = true
		if group.MinSelections < 0 || group.MaxSelections < 1 ||
			group.MinSelections > group.MaxSelections {
			return "Revisa los mínimos y máximos de cada grupo."
		}
		if group.Required && group.MinSelections < 1 {
			return "Un grupo obligatorio debe exigir al menos una selección."
		}
		if group.MaxSelections > len(group.Options) {
			return "El máximo de selecciones no puede superar las opciones disponibles."
		}

		optionNames := map[string]bool{}
		for _, option := range group.Options {
			optionName := strings.TrimSpace(option.Name)
			if optionName == "" || len([]rune(optionName)) > 80 {
				return "Cada opción necesita un nombre de hasta 80 caracteres."
			}
			optionKey := strings.ToLower(optionName)
			if optionNames[optionKey] {
				return "Las opciones de un mismo grupo necesitan nombres distintos."
			}
			optionNames[optionKey] = true
			value := strings.TrimSpace(option.Surcharge)
			if value == "" {
				value = "0"
			}
			surcharge, err := strconv.ParseFloat(value, 64)
			if err != nil || surcharge < 0 {
				return "Cada recargo debe ser un importe válido mayor o igual a cero."
			}
		}
	}
	return ""
}

func (a *API) productSupportsModifiers(
	ctx context.Context,
	q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	organizationID string,
	productID string,
) (bool, error) {
	var exists, combo bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
		         SELECT 1 FROM products
		         WHERE id=$1 AND organization_id=$2
		       ),
		       EXISTS(
		         SELECT 1 FROM menu_combos
		         WHERE product_id=$1 AND organization_id=$2
		       )
	`, productID, organizationID).Scan(&exists, &combo)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, pgx.ErrNoRows
	}
	return !combo, nil
}

func (a *API) getProductModifiers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	productID := r.PathValue("id")
	supported, err := a.productSupportsModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos validar el producto.")
		return
	}
	if !supported {
		fail(w, 409, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
		return
	}
	out, err := loadProductModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos cargar los modificadores.")
		return
	}
	writeJSON(w, 200, out)
}

func (a *API) saveProductModifiers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	productID := r.PathValue("id")
	var in modifierConfigInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_modifiers", "Revisa la configuración de modificadores.")
		return
	}
	if message := validateModifierConfig(in); message != "" {
		fail(w, 400, "invalid_modifiers", message)
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos iniciar la actualización.")
		return
	}
	defer tx.Rollback(r.Context())

	supported, err := a.productSupportsModifiers(r.Context(), tx, s.OrganizationID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos validar el producto.")
		return
	}
	if !supported {
		fail(w, 409, "modifiers_not_supported_for_combo", "Los combos administran sus opciones desde Menús y combos.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		DELETE FROM product_modifier_groups
		WHERE organization_id=$1 AND product_id=$2
	`, s.OrganizationID, productID); err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos reemplazar los modificadores.")
		return
	}

	for groupIndex, group := range in.Groups {
		var groupID string
		err = tx.QueryRow(r.Context(), `
			INSERT INTO product_modifier_groups(
			  organization_id,product_id,name,required,min_selections,max_selections,sort_order
			)
			VALUES($1,$2,$3,$4,$5,$6,$7)
			RETURNING id
		`, s.OrganizationID, productID, strings.TrimSpace(group.Name),
			group.Required, group.MinSelections, group.MaxSelections, groupIndex).Scan(&groupID)
		if err != nil {
			fail(w, 409, "modifier_group_conflict", "No pudimos guardar un grupo de modificadores.")
			return
		}
		for optionIndex, option := range group.Options {
			surcharge := strings.TrimSpace(option.Surcharge)
			if surcharge == "" {
				surcharge = "0"
			}
			if _, err = tx.Exec(r.Context(), `
				INSERT INTO product_modifier_options(
				  organization_id,group_id,name,surcharge,sort_order
				)
				VALUES($1,$2,$3,$4,$5)
			`, s.OrganizationID, groupID, strings.TrimSpace(option.Name),
				surcharge, optionIndex); err != nil {
				fail(w, 409, "modifier_option_conflict", "No pudimos guardar una opción de modificador.")
				return
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "modifiers_unavailable", "No pudimos confirmar los modificadores.")
		return
	}

	a.audit(r, "product.modifiers_saved", "product", productID)
	out, err := loadProductModifiers(r.Context(), a.db, s.OrganizationID, productID)
	if err != nil {
		fail(w, 503, "modifiers_unavailable", "Los modificadores se guardaron, pero no pudimos recargarlos.")
		return
	}
	writeJSON(w, 200, out)
}
