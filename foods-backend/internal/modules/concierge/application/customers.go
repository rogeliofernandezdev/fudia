package application

import (
	"context"
	"strings"

	customercontracts "github.com/foods-platform/foods-backend/internal/modules/customers/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// CustomerLookup lleva el número en el cuerpo, nunca en la URL.
type CustomerLookup struct {
	CustomerPhone string `json:"customerPhone"`
}

// CustomerProfile identifica al comensal en la empresa del QR para que
// Concierge salude por su nombre o lo pida si es nuevo, y sepa si ya respondió
// la pregunta promocional.
func (a *Concierge) CustomerProfile(ctx context.Context, token string, in CustomerLookup) (customercontracts.WhatsAppProfile, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return customercontracts.WhatsAppProfile{}, err
	}
	if a.customers == nil {
		return customercontracts.WhatsAppProfile{}, conciergeFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos consultar al cliente.")
	}
	return a.customers.WhatsAppProfile(ctx, qr.Scope, strings.TrimSpace(in.CustomerPhone))
}

// RecordMarketingConsent guarda la pregunta promocional o la respuesta del
// cliente en la empresa del QR, con su texto literal como evidencia.
func (a *Concierge) RecordMarketingConsent(ctx context.Context, token string, in customercontracts.ConsentEvidence) (customercontracts.WhatsAppProfile, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return customercontracts.WhatsAppProfile{}, err
	}
	if a.customers == nil {
		return customercontracts.WhatsAppProfile{}, conciergeFailure(apperror.Unavailable, "concierge_unavailable", "No pudimos registrar la preferencia del cliente.")
	}
	return a.customers.RecordWhatsAppConsent(ctx, qr.Scope, in)
}
