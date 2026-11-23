package domain

import (
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Motorizado, liberación, alertas de recogida y entrega fallida de Delivery
// Fudia. No hay estados nuevos: el despacho sigue en accepted o picked_up y estos
// hechos se guardan como columnas del reparto.

// Acciones del panel de la central.
const (
	// ActionRelease libera el pedido antes de recogerlo: vuelve a ofrecerse a las demás centrales.
	ActionRelease = "release"
	// ActionKeep mantiene al motorizado ante una alerta de recogida.
	ActionKeep = "keep"
	// ActionFail registra la entrega fallida después del reporte del motorizado.
	ActionFail = "fail"
)

// ActionNotDelivered: el motorizado marca «No pude entregar» en su enlace.
const ActionNotDelivered = "not_delivered"

// Resultados de las acciones nuevas.
const (
	OutcomeReleased        = "released"
	OutcomeKept            = "kept"
	OutcomeDeliveryFailed  = "delivery_failed"
	OutcomeAlreadyPickedUp = "already_picked_up"
	OutcomeNotReported     = "not_reported"
	OutcomeTooSoon         = "too_soon"
	OutcomeCourierAssigned = "courier_assigned"
	OutcomeIssueReported   = "issue_reported"
)

// Alertas de recogida: el pedido está listo y el motorizado no llega. Nunca
// liberan el pedido por tiempo; la central decide.
const (
	PickupAlertWarning  = "warning"
	PickupAlertUrgent   = "urgent"
	PickupAlertCritical = "critical"
)

var pickupAlertSteps = []struct {
	after time.Duration
	name  string
}{
	{8 * time.Minute, PickupAlertCritical},
	{5 * time.Minute, PickupAlertUrgent},
	{3 * time.Minute, PickupAlertWarning},
}

// PickupCriticalAfter es el nivel crítico de la alerta de la central.
const PickupCriticalAfter = 8 * time.Minute

// PickupAlertLevel devuelve el nivel (0 a 3) y su nombre para la espera dada.
func PickupAlertLevel(wait time.Duration) (int, string) {
	for i, step := range pickupAlertSteps {
		if wait >= step.after {
			return len(pickupAlertSteps) - i, step.name
		}
	}
	return 0, ""
}

// Límites de la entrega fallida.
const (
	// FailedDeliveryMinWait: el cliente tiene este tiempo para responder al chat
	// después del reporte del motorizado antes de registrar la entrega fallida.
	FailedDeliveryMinWait = 10 * time.Minute
	// FailedDeliveryMaxDistanceMeters: el motorizado debe reportar cerca del destino.
	FailedDeliveryMaxDistanceMeters = 300
	// FailedDeliveryMonthlyLimit: entregas fallidas con aporte de Fudia por central en 30 días.
	FailedDeliveryMonthlyLimit = 3
	// FailedDeliveryFudiaCap: aporte máximo de Fudia por incidencia, en la moneda del local.
	FailedDeliveryFudiaCap = 100.0
)

// FailedDeliveryFacts es lo que se sabe del reparto al registrar la entrega fallida.
type FailedDeliveryFacts struct {
	FoodAmount float64
	// DistanceMeters: distancia del motorizado al destino al reportar; nil si no compartía ubicación.
	DistanceMeters *int
	// DestinationKnown: el pedido tiene las coordenadas del punto de entrega.
	DestinationKnown bool
	CustomerNotified bool
	// CoveredRecently: entregas fallidas de la central con aporte de Fudia en 30 días.
	CoveredRecently int
}

// FailedDeliverySettlement reparte la pérdida: la central reembolsa al motorizado
// el 100 % de la comida y luego Fudia y la central asumen 50 % cada una. Fudia
// aporta solo si la incidencia se valida y está dentro de los límites; si no, la
// central asume el total.
type FailedDeliverySettlement struct {
	FoodAmount, CourierReimbursement, PartnerShare, FudiaShare float64
	FudiaShareEligible                                         bool
	ValidationNotes                                            []string
}

func cents(value float64) float64 { return math.Round(value*100) / 100 }

// SettleFailedDelivery valida la incidencia y calcula los montos.
func SettleFailedDelivery(f FailedDeliveryFacts) FailedDeliverySettlement {
	food := cents(math.Max(0, f.FoodAmount))
	out := FailedDeliverySettlement{FoodAmount: food, CourierReimbursement: food, PartnerShare: food, ValidationNotes: []string{}}
	eligible := true
	switch {
	case f.DistanceMeters == nil:
		eligible = false
		out.ValidationNotes = append(out.ValidationNotes, "El motorizado no compartía su ubicación al reportar.")
	case !f.DestinationKnown:
		eligible = false
		out.ValidationNotes = append(out.ValidationNotes, "El pedido no tiene la ubicación del punto de entrega.")
	case *f.DistanceMeters > FailedDeliveryMaxDistanceMeters:
		eligible = false
		out.ValidationNotes = append(out.ValidationNotes, "El motorizado reportó lejos del punto de entrega.")
	}
	if !f.CustomerNotified {
		eligible = false
		out.ValidationNotes = append(out.ValidationNotes, "No se pudo avisar al cliente por el chat.")
	}
	if f.CoveredRecently >= FailedDeliveryMonthlyLimit {
		eligible = false
		out.ValidationNotes = append(out.ValidationNotes, "La central superó el límite de entregas fallidas cubiertas en 30 días.")
	}
	if !eligible || food == 0 {
		return out
	}
	out.FudiaShareEligible = true
	out.FudiaShare = cents(math.Min(food/2, FailedDeliveryFudiaCap))
	out.PartnerShare = cents(food - out.FudiaShare)
	if out.FudiaShare < food/2 {
		out.ValidationNotes = append(out.ValidationNotes, "El aporte de Fudia llegó a su tope por incidencia.")
	}
	return out
}

var courierTokenPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// ValidCourierToken valida la forma del enlace del motorizado antes de consultar la base.
func ValidCourierToken(token string) bool { return courierTokenPattern.MatchString(token) }

// NormalizeCourierName limpia el nombre del motorizado; ok=false si no es válido.
func NormalizeCourierName(name string) (string, bool) {
	name = strings.Join(strings.Fields(name), " ")
	n := utf8.RuneCountInString(name)
	return name, n >= 2 && n <= 60
}

// ValidCourierAction acepta las acciones del enlace del motorizado.
func ValidCourierAction(action string) bool {
	return action == ActionPickup || action == ActionDeliver || action == ActionNotDelivered
}

// CourierInput asigna o cambia el motorizado de un reparto aceptado.
type CourierInput struct {
	Token       string `json:"token"`
	CourierName string `json:"courierName"`
}

// CourierResult devuelve el enlace vigente del motorizado.
type CourierResult struct {
	Outcome      string `json:"outcome"`
	CourierName  string `json:"courierName,omitempty"`
	CourierToken string `json:"courierToken,omitempty"`
}

// CourierView es lo que ve el motorizado al abrir su enlace.
type CourierView struct {
	PartnerName string     `json:"partnerName"`
	CourierName string     `json:"courierName"`
	Order       PanelOrder `json:"order"`
}

// FailedDelivery es una entrega fallida registrada, con sus montos.
type FailedDelivery struct {
	OrderCode            string   `json:"orderCode"`
	CustomerName         string   `json:"customerName"`
	Address              string   `json:"address"`
	CourierName          string   `json:"courierName"`
	CurrencySymbol       string   `json:"currencySymbol"`
	FoodAmount           string   `json:"foodAmount"`
	CourierReimbursement string   `json:"courierReimbursement"`
	PartnerShare         string   `json:"partnerShare"`
	FudiaShare           string   `json:"fudiaShare"`
	FudiaShareEligible   bool     `json:"fudiaShareEligible"`
	ValidationNotes      []string `json:"validationNotes"`
	RegisteredAt         string   `json:"registeredAt"`
}
