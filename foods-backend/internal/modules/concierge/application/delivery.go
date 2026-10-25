package application

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	deliverycontracts "github.com/foods-platform/foods-backend/internal/modules/delivery/contracts"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// DeliverySetup son los datos del local que definen recojo y delivery.
type DeliverySetup struct {
	LocationName, Address, Phone, CurrencySymbol string
	Latitude, Longitude                          *float64
	Tiers                                        []deliverycontracts.FeeTier
}

func (d DeliverySetup) deliveryAvailable() bool {
	return d.Latitude != nil && d.Longitude != nil && len(d.Tiers) > 0
}

// DeliveryOrderInput es el pedido de recojo o delivery que confirma el cliente.
// QuotedFee es la tarifa que el cliente aceptó; si cambió, el pedido se rechaza.
type DeliveryOrderInput struct {
	Fulfillment    string                          `json:"fulfillment"`
	CustomerName   string                          `json:"customerName"`
	CustomerPhone  string                          `json:"customerPhone"`
	ConversationID string                          `json:"conversationId"`
	RequestID      string                          `json:"requestId"`
	Address        string                          `json:"address"`
	Reference      string                          `json:"reference"`
	Notes          string                          `json:"notes"`
	Latitude       *float64                        `json:"latitude"`
	Longitude      *float64                        `json:"longitude"`
	QuotedFee      string                          `json:"quotedFee"`
	Items          []ordercontracts.OrderItemInput `json:"items"`
}

func money(value float64) string { return fmt.Sprintf("%.2f", value) }

func tierViews(tiers []deliverycontracts.FeeTier) []domain.FeeTierView {
	out := []domain.FeeTierView{}
	for _, tier := range deliverycontracts.SortedTiers(tiers) {
		out = append(out, domain.FeeTierView{MaxDistanceKm: money(tier.MaxDistanceKm), Fee: money(tier.Fee)})
	}
	return out
}

func (a *Concierge) deliverySetup(ctx context.Context, qr QRContext) (DeliverySetup, error) {
	setup, err := a.repo.DeliverySetup(ctx, qr.Scope)
	if err != nil {
		return setup, conciergeFailure(apperror.Unavailable, "delivery_unavailable", "No pudimos cargar las opciones de entrega.")
	}
	setup.CurrencySymbol = qr.CurrencySymbol
	setup.LocationName = qr.LocationName
	return setup, nil
}

func (a *Concierge) DeliveryOptions(ctx context.Context, token string) (domain.DeliveryOptions, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return domain.DeliveryOptions{}, err
	}
	setup, err := a.deliverySetup(ctx, qr)
	if err != nil {
		return domain.DeliveryOptions{}, err
	}
	out := domain.DeliveryOptions{
		LocationName:      qr.LocationName,
		LocationAddress:   setup.Address,
		LocationPhone:     setup.Phone,
		CurrencySymbol:    qr.CurrencySymbol,
		PickupAvailable:   true,
		DeliveryAvailable: setup.deliveryAvailable(),
		MaxDistanceKm:     money(0),
		Tiers:             []domain.FeeTierView{},
	}
	if out.DeliveryAvailable {
		out.MaxDistanceKm = money(deliverycontracts.CoverageKm(setup.Tiers))
		out.Tiers = tierViews(setup.Tiers)
	}
	return out, nil
}

type quoteResult struct {
	view     domain.DeliveryQuote
	fee      float64
	distance float64
}

func quote(setup DeliverySetup, latitude, longitude *float64) (quoteResult, error) {
	var out quoteResult
	if !setup.deliveryAvailable() {
		return out, conciergeFailure(apperror.Conflict, "delivery_unavailable", "Este local no tiene delivery configurado; puedes recoger tu pedido en el local.")
	}
	if latitude == nil || longitude == nil || !deliverycontracts.ValidCoordinates(*latitude, *longitude) {
		return out, conciergeFailure(apperror.Invalid, "invalid_location", "Comparte una ubicación válida para calcular el delivery.")
	}
	out.distance = deliverycontracts.DistanceKm(*setup.Latitude, *setup.Longitude, *latitude, *longitude)
	tier, covered := deliverycontracts.QuoteFee(setup.Tiers, out.distance)
	out.fee = tier.Fee
	out.view = domain.DeliveryQuote{
		Covered:        covered,
		DistanceKm:     money(out.distance),
		Fee:            money(tier.Fee),
		MaxDistanceKm:  money(deliverycontracts.CoverageKm(setup.Tiers)),
		CurrencySymbol: setup.CurrencySymbol,
	}
	return out, nil
}

func (a *Concierge) QuoteDelivery(ctx context.Context, token string, in domain.DeliveryQuoteInput) (domain.DeliveryQuote, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return domain.DeliveryQuote{}, err
	}
	setup, err := a.deliverySetup(ctx, qr)
	if err != nil {
		return domain.DeliveryQuote{}, err
	}
	out, err := quote(setup, in.Latitude, in.Longitude)
	return out.view, err
}

// QuoteAdminDelivery sugiere el costo de envío del alta manual con las mismas
// tarifas del local que usa Concierge. El cajero puede corregirlo al guardar.
func (a *Concierge) QuoteAdminDelivery(ctx context.Context, s scope, in domain.DeliveryQuoteInput) (domain.DeliveryQuote, error) {
	setup, err := a.repo.DeliverySetup(ctx, s)
	if err != nil {
		return domain.DeliveryQuote{}, conciergeFailure(apperror.Unavailable, "delivery_unavailable", "No pudimos cargar las tarifas de delivery.")
	}
	out, err := quote(setup, in.Latitude, in.Longitude)
	return out.view, err
}

// SubmitDelivery crea el pedido de recojo o delivery. La tarifa se recalcula
// con las tarifas vigentes del local; nunca se acepta un costo del cliente.
func (a *Concierge) SubmitDelivery(ctx context.Context, token string, in DeliveryOrderInput) (ordercontracts.TableSubmission, error) {
	var result ordercontracts.TableSubmission
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return result, err
	}
	in.Fulfillment = strings.TrimSpace(in.Fulfillment)
	if in.Fulfillment != "delivery" && in.Fulfillment != "pickup" {
		return result, conciergeFailure(apperror.Invalid, "invalid_fulfillment", "Indica si el pedido es para delivery o recojo en el local.")
	}
	// Un reintento devuelve el pedido creado aunque las tarifas hayan cambiado.
	if existing, found, replayErr := a.orders.ConciergeDeliveryReplay(ctx, qr.Scope, in.RequestID); replayErr != nil {
		return result, replayErr
	} else if found {
		return ordercontracts.TableSubmission{Order: existing}, nil
	}
	order := ordercontracts.ConciergeDeliveryInput{
		Channel:        "recojo",
		CustomerName:   in.CustomerName,
		CustomerPhone:  in.CustomerPhone,
		ConversationID: in.ConversationID,
		RequestID:      in.RequestID,
		Notes:          in.Notes,
		Items:          in.Items,
	}
	if in.Fulfillment == "delivery" {
		setup, setupErr := a.deliverySetup(ctx, qr)
		if setupErr != nil {
			return result, setupErr
		}
		quoted, quoteErr := quote(setup, in.Latitude, in.Longitude)
		if quoteErr != nil {
			return result, quoteErr
		}
		if !quoted.view.Covered {
			return result, conciergeFailure(apperror.Conflict, "delivery_out_of_range",
				fmt.Sprintf("La ubicación está a %s km y el delivery llega hasta %s km.", quoted.view.DistanceKm, quoted.view.MaxDistanceKm))
		}
		accepted, parseErr := strconv.ParseFloat(strings.TrimSpace(in.QuotedFee), 64)
		if parseErr != nil {
			return result, conciergeFailure(apperror.Invalid, "quoted_fee_required", "Confirma el costo de envío antes de enviar el pedido.")
		}
		if math.Abs(accepted-quoted.fee) >= 0.005 {
			return result, conciergeFailure(apperror.Conflict, "delivery_fee_changed",
				fmt.Sprintf("El costo de envío cambió a %s %s.", qr.CurrencySymbol, quoted.view.Fee))
		}
		distance := quoted.distance
		order.Channel = "delivery"
		order.Address = in.Address
		order.Reference = in.Reference
		order.Latitude = in.Latitude
		order.Longitude = in.Longitude
		order.DistanceKm = &distance
		order.DeliveryFee = quoted.fee
	}
	return a.orders.SubmitConciergeDelivery(ctx, qr.Scope, order)
}

func (a *Concierge) OrderStatus(ctx context.Context, token, orderID, conversationID string) (domain.OrderStatusView, error) {
	qr, err := a.resolve(ctx, token)
	if err != nil {
		return domain.OrderStatusView{}, err
	}
	state, err := a.orders.ConciergeOrder(ctx, qr.Scope, conversationID, orderID)
	if err != nil {
		return domain.OrderStatusView{}, err
	}
	o := state.Order
	return domain.OrderStatusView{
		OrderID:       o.ID,
		Code:          o.Code,
		Channel:       o.Channel,
		Status:        o.Status,
		Subtotal:      o.Subtotal,
		DeliveryFee:   o.DeliveryFee,
		Total:         o.Total,
		PaymentStatus: o.PaymentStatus,
		Active:        state.Active,
	}, nil
}

func (a *Concierge) DeliveryFees(ctx context.Context, s scope) (domain.DeliveryFeesView, error) {
	out, tiers, err := a.repo.DeliveryFees(ctx, s)
	if err != nil {
		return out, conciergeFailure(apperror.Unavailable, "delivery_fees_unavailable", "No pudimos cargar las tarifas de delivery.")
	}
	out.Tiers = tierViews(tiers)
	return out, nil
}

func (a *Concierge) SaveDeliveryFees(ctx context.Context, s scope, in domain.DeliveryFeesInput) (domain.DeliveryFeesView, error) {
	tiers := make([]deliverycontracts.FeeTier, 0, len(in.Tiers))
	for _, tier := range in.Tiers {
		tiers = append(tiers, deliverycontracts.FeeTier{MaxDistanceKm: tier.MaxDistanceKm, Fee: tier.Fee})
	}
	if message := deliverycontracts.ValidateFeeTiers(tiers); message != "" {
		return domain.DeliveryFeesView{}, conciergeFailure(apperror.Invalid, "invalid_delivery_fees", message)
	}
	if err := a.repo.ReplaceDeliveryFees(ctx, s, deliverycontracts.SortedTiers(tiers)); err != nil {
		return domain.DeliveryFeesView{}, conciergeFailure(apperror.Unavailable, "delivery_fees_unavailable", "No pudimos guardar las tarifas de delivery.")
	}
	_ = a.audit.Record(ctx, s, "concierge.delivery_fees.updated", "location", s.LocationID)
	return a.DeliveryFees(ctx, s)
}
