// Package fake is a deterministic provider for development and tests. It
// accepts every document, except customer names containing "[rechazar]"
// (rejected) or "[error]" (temporary failure, retried).
package fake

import (
	"context"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
)

type Provider struct{}

var _ ports.Provider = Provider{}

func New() Provider                                        { return Provider{} }
func (Provider) Key() string                               { return "fake" }
func (Provider) Label() string                             { return "Simulado (pruebas)" }
func (Provider) TestOnly() bool                            { return true }
func (Provider) CredentialFields() []ports.CredentialField { return []ports.CredentialField{} }
func (Provider) RequiresCredentials(string) bool           { return false }
func (Provider) ValidateCredentials(string, map[string]string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (Provider) Issue(_ context.Context, document ports.Document) (ports.Result, error) {
	name := strings.ToLower(document.Customer.Name)
	switch {
	case strings.Contains(name, "[error]"):
		return ports.Result{}, errors.New("proveedor simulado no disponible")
	case strings.Contains(name, "[rechazar]"):
		return ports.Result{Status: domain.StatusRejected, Code: "2800", Message: "Rechazo simulado por el proveedor de pruebas."}, nil
	}
	return ports.Result{Status: domain.StatusAccepted, Reference: "SIM-" + document.FormattedNumber, Code: "0", Message: "Aceptado por el proveedor simulado."}, nil
}
