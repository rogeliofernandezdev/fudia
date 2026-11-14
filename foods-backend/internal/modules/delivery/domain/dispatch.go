package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Delivery Fudia: la plataforma trabaja con centrales de reparto en un orden
// global y activa Delivery Fudia por empresa; todos sus locales lo usan. Un
// pedido confirmado se ofrece a la 1.ª central y, si rechaza o no responde, pasa
// a la siguiente.

// Modos de delivery de una empresa.
const (
	// ModeOwn: la empresa reparte con su propio delivery; Fudia solo recibe el pedido.
	ModeOwn = "own"
	// ModeFudia: el pedido se ofrece a las centrales de la plataforma.
	ModeFudia = "fudia"
)

// OfferTimeout es cuánto espera una central antes de pasar a la siguiente.
const OfferTimeout = 3 * time.Minute

// UndeliveredOfferTimeout descarta una oferta cuyo mensaje nunca se pudo enviar.
const UndeliveredOfferTimeout = 5 * time.Minute

// Estados del despacho.
const (
	DispatchSearching  = "searching"
	DispatchAccepted   = "accepted"
	DispatchUnassigned = "unassigned"
	DispatchPickedUp   = "picked_up"
	DispatchOwn        = "own"
	DispatchCancelled  = "cancelled"
)

// Estados de una oferta a una central.
const (
	OfferPending   = "pending"
	OfferAccepted  = "accepted"
	OfferRejected  = "rejected"
	OfferExpired   = "expired"
	OfferWithdrawn = "withdrawn"
)

// Mensajes a la central.
const (
	MessageOffer     = "offer"
	MessageReady     = "ready"
	MessageCancelled = "cancelled"
)

// Acciones de los botones de WhatsApp de la central.
const (
	ActionAccept = "accept"
	ActionReject = "reject"
	ActionPickup = "pickup"
)

// Resultados de una acción; Concierge redacta la respuesta a la central.
const (
	OutcomeAccepted        = "accepted"
	OutcomeAlreadyAccepted = "already_accepted"
	OutcomeRejected        = "rejected"
	OutcomeTaken           = "taken"
	OutcomeExpired         = "expired"
	OutcomeCancelled       = "cancelled"
	OutcomeNotReady        = "not_ready"
	OutcomePickedUp        = "picked_up"
	OutcomeNotFound        = "not_found"
	OutcomePaymentConflict = "payment_conflict"
)

// Partner es una central de delivery registrada por la plataforma.
type Partner struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	WhatsAppPhone  string   `json:"whatsappPhone"`
	Active         bool     `json:"active"`
	CoverageCodes  []string `json:"coverageCodes"`
	CoverageLabels []string `json:"coverageLabels"`
	// Priority es el orden global: la 1 recibe primero cada pedido.
	Priority int `json:"priority"`
	// PanelToken abre el panel web de la central; solo lo ve Plataforma.
	PanelToken string `json:"panelToken"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// PartnerInput crea o edita una central.
type PartnerInput struct {
	Name          string   `json:"name"`
	WhatsAppPhone string   `json:"whatsappPhone"`
	Active        *bool    `json:"active"`
	CoverageCodes []string `json:"coverageCodes"`
}

// PartnerOrderInput fija el orden global de todas las centrales.
type PartnerOrderInput struct {
	PartnerIDs []string `json:"partnerIds"`
}

// OrganizationDelivery indica si la empresa usa Delivery Fudia en sus locales.
type OrganizationDelivery struct {
	OrganizationID string `json:"organizationId"`
	Enabled        bool   `json:"enabled"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

// OrganizationDeliveryInput activa o desactiva Delivery Fudia para la empresa.
type OrganizationDeliveryInput struct {
	Enabled bool `json:"enabled"`
}

// DeliveryLink es el enlace/QR de delivery del local: abre el WhatsApp de
// Fudia ya ligado al local para pedir delivery o recojo. Token vacío: aún no
// se generó.
type DeliveryLink struct {
	LocationID string `json:"locationId"`
	Token      string `json:"token"`
	Active     bool   `json:"active"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

// DispatchOffer es un intento con una central, para el historial del pedido.
type DispatchOffer struct {
	PartnerName string `json:"partnerName"`
	Round       int    `json:"round"`
	Priority    int    `json:"priority"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	RespondedAt string `json:"respondedAt,omitempty"`
}

// Dispatch es el despacho de un pedido a las centrales.
type Dispatch struct {
	OrderID     string          `json:"orderId"`
	Status      string          `json:"status"`
	Round       int             `json:"round"`
	PartnerName string          `json:"partnerName,omitempty"`
	AcceptedAt  string          `json:"acceptedAt,omitempty"`
	PickedUpAt  string          `json:"pickedUpAt,omitempty"`
	DeliveredAt string          `json:"deliveredAt,omitempty"`
	Offers      []DispatchOffer `json:"offers"`
}

// MessageItem es una línea del pedido en el mensaje a la central.
type MessageItem struct {
	Quantity float64 `json:"quantity"`
	Name     string  `json:"name"`
	Note     string  `json:"note,omitempty"`
}

// PartnerMessage es un mensaje reclamado por Concierge para la central. Viajan
// hechos del pedido; Concierge redacta el texto y arma los botones con
// ActionToken (Aceptar/Rechazar en la oferta, Recoger cuando está listo).
type PartnerMessage struct {
	RoutingKey       string        `json:"channelKey"`
	ID               string        `json:"id"`
	Kind             string        `json:"kind"`
	PartnerName      string        `json:"partnerName"`
	PartnerPhone     string        `json:"partnerPhone"`
	ActionToken      string        `json:"actionToken"`
	ExpiresInSeconds int           `json:"expiresInSeconds"`
	OrderCode        string        `json:"orderCode"`
	OrganizationName string        `json:"organizationName"`
	LocationName     string        `json:"locationName"`
	LocationAddress  string        `json:"locationAddress"`
	CustomerName     string        `json:"customerName"`
	CustomerPhone    string        `json:"customerPhone"`
	Address          string        `json:"address"`
	Reference        string        `json:"reference"`
	Latitude         *float64      `json:"latitude"`
	Longitude        *float64      `json:"longitude"`
	DistanceKm       *float64      `json:"distanceKm"`
	Notes            string        `json:"notes"`
	Items            []MessageItem `json:"items"`
	CurrencySymbol   string        `json:"currencySymbol"`
	Subtotal         string        `json:"subtotal"`
	DeliveryFee      string        `json:"deliveryFee"`
	Total            string        `json:"total"`
	CreatedAt        string        `json:"createdAt"`
}

// ActionInput llega de Concierge cuando la central pulsa un botón.
type ActionInput struct {
	Action string `json:"action"`
	Token  string `json:"token"`
	// Phone es el WhatsApp que pulsó el botón; debe ser el de la central.
	Phone string `json:"phone"`
}

// ActionResult indica qué pasó para que Concierge responda a la central.
type ActionResult struct {
	Outcome          string `json:"outcome"`
	OrderCode        string `json:"orderCode,omitempty"`
	OrganizationName string `json:"organizationName,omitempty"`
	LocationName     string `json:"locationName,omitempty"`
}

// NormalizePhone deja solo los dígitos de un WhatsApp con código de país.
func NormalizePhone(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidPhone exige código de país: 8 a 15 dígitos sin cero inicial (E.164).
func ValidPhone(digits string) bool {
	if len(digits) < 8 || len(digits) > 15 || digits[0] == '0' {
		return false
	}
	return NormalizePhone(digits) == digits
}

// NormalizePartner valida y limpia los datos de una central. Devuelve el
// mensaje de error para el usuario, vacío si es válida.
func NormalizePartner(in PartnerInput) (PartnerInput, string) {
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	in.WhatsAppPhone = NormalizePhone(in.WhatsAppPhone)
	if n := utf8.RuneCountInString(in.Name); n < 2 || n > 120 {
		return in, "Escribe el nombre de la central (2 a 120 caracteres)."
	}
	if !ValidPhone(in.WhatsAppPhone) {
		return in, "Escribe el WhatsApp con código de país, por ejemplo 51 987654321."
	}
	if len(in.CoverageCodes) == 0 && in.Active != nil && !*in.Active {
		return in, ""
	}
	if len(in.CoverageCodes) == 0 || len(in.CoverageCodes) > 2000 {
		return in, "Marca al menos un distrito de atención."
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(in.CoverageCodes))
	for _, code := range in.CoverageCodes {
		code = strings.TrimSpace(code)
		if len(code) != 6 || NormalizePhone(code) != code {
			return in, "Selecciona un distrito de atención válido."
		}
		if !seen[code] {
			clean = append(clean, code)
			seen[code] = true
		}
	}
	in.CoverageCodes = clean
	return in, ""
}

// NormalizeOrder limpia el orden global: sin vacíos ni centrales repetidas.
func NormalizeOrder(in PartnerOrderInput) ([]string, string) {
	seen := map[string]bool{}
	ids := make([]string, 0, len(in.PartnerIDs))
	for _, id := range in.PartnerIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if seen[id] {
			return nil, "Cada central aparece una sola vez en el orden."
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, "Indica el orden de las centrales."
	}
	return ids, ""
}

// ValidAction acepta solo los botones conocidos.
func ValidAction(action string) bool {
	return action == ActionAccept || action == ActionReject || action == ActionPickup
}

// OrderDispatchable indica si un pedido confirmado debe ofrecerse a las centrales.
func OrderDispatchable(channel, status string, completed bool) bool {
	if channel != "delivery" || completed {
		return false
	}
	return status == "confirmado" || status == "preparando" || status == "listo"
}

// ActiveDispatch indica si el despacho aún puede cambiar por una central.
func ActiveDispatch(status string) bool {
	return status == DispatchSearching || status == DispatchAccepted || status == DispatchUnassigned
}

// Retryable indica si caja puede volver a ofrecer el pedido desde la 1.ª central.
func Retryable(status string) bool {
	return status == DispatchUnassigned || status == DispatchOwn
}
