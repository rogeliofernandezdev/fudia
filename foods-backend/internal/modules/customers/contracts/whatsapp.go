// Package contracts publica lo que Concierge necesita de Clientes: reconocer al
// comensal por su número y registrar su consentimiento promocional.
package contracts

// Estados del consentimiento promocional de un cliente en su empresa. pending
// significa que nunca se le preguntó; asked, que se le preguntó sin respuesta.
const (
	ConsentPending   = "pending"
	ConsentAsked     = "asked"
	ConsentGranted   = "granted"
	ConsentDeclined  = "declined"
	ConsentWithdrawn = "withdrawn"
)

// ValidConsentDecision acepta los eventos que Concierge puede registrar.
func ValidConsentDecision(decision string) bool {
	return decision == ConsentAsked || decision == ConsentGranted || decision == ConsentDeclined || decision == ConsentWithdrawn
}

// WhatsAppProfile identifica a un comensal de WhatsApp dentro de una empresa.
type WhatsAppProfile struct {
	Known            bool   `json:"known"`
	CustomerName     string `json:"customerName"`
	MarketingConsent string `json:"marketingConsent"`
}

// ConsentEvidence es una pregunta o respuesta del consentimiento. Evidence
// guarda el texto literal del cliente; RequestID hace idempotente el registro.
type ConsentEvidence struct {
	CustomerPhone  string `json:"customerPhone"`
	ConversationID string `json:"conversationId"`
	RequestID      string `json:"requestId"`
	Decision       string `json:"decision"`
	Evidence       string `json:"evidence"`
}
