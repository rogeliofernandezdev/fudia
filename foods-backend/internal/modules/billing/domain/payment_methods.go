package domain

import (
	"regexp"
	"strings"
)

type PaymentMethodView struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Active          bool   `json:"active"`
	SalesEnabled    bool   `json:"salesEnabled"`
	ExpensesEnabled bool   `json:"expensesEnabled"`
	AffectsCash     bool   `json:"affectsCash"`
	SortOrder       int    `json:"sortOrder"`
}
type PaymentMethodInput struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	SalesEnabled    bool   `json:"salesEnabled"`
	ExpensesEnabled bool   `json:"expensesEnabled"`
	AffectsCash     bool   `json:"affectsCash"`
	SortOrder       int    `json:"sortOrder"`
}

var PaymentMethodCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// Inactive entries retain their history and still consume a catalog slot.
const MaxPaymentMethods = 4

// Retained for activation checks on catalogs created before the total limit.
const MaxActivePaymentMethods = MaxPaymentMethods

func NormalizePaymentMethodInput(in *PaymentMethodInput) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
}
func ValidPaymentMethodInput(in PaymentMethodInput, creating bool) bool {
	if creating && !PaymentMethodCodePattern.MatchString(in.Code) {
		return false
	}
	if in.Name == "" || len(in.Name) > 80 || len(in.Description) > 180 || in.SortOrder < 0 || in.SortOrder > 9999 {
		return false
	}
	return in.SalesEnabled || in.ExpensesEnabled
}
