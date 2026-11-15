package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// Panel web del proveedor de Delivery Fudia. La sesión del operador identifica
// a la central; cada acción vuelve a comprobar que la oferta es suya.

// PanelPartner es la central de la sesión del operador.
type PanelPartner struct {
	ID, Name string
	Active   bool
}

// PanelRepository lee y actualiza lo que la central ve en su panel.
type PanelRepository interface {
	// StartOfferTimers inicia los 3 minutos de las ofertas que el panel muestra por primera vez.
	StartOfferTimers(ctx context.Context, partnerID string) error
	PanelOrders(ctx context.Context, partnerID string) ([]domain.PanelOrder, error)
	Tracking(ctx context.Context, s scope, orderID string) (domain.Tracking, error)
	// FailedDeliveries lista las entregas fallidas recientes de la central.
	FailedDeliveries(ctx context.Context, partnerID string) ([]domain.FailedDelivery, error)
	// CourierView carga el reparto vigente del enlace del motorizado.
	CourierView(ctx context.Context, courierToken string) (domain.CourierView, error)
	// SetCourierLocationByToken guarda la ubicación en el reparto del motorizado; false si el enlace ya no vale.
	SetCourierLocationByToken(ctx context.Context, courierToken string, latitude, longitude float64) (bool, error)
	// Opción «Delivery» del restaurante (alcance de la sesión).
	BoardOrders(ctx context.Context, s scope) ([]domain.BoardOrder, error)
	LocationPartners(ctx context.Context, s scope) (domain.LocationPartners, error)
	LocationFailedDeliveries(ctx context.Context, s scope) ([]domain.LocationFailedDelivery, error)
}

// WithPanel conecta las consultas del panel web.
func (a *Dispatcher) WithPanel(panel PanelRepository) *Dispatcher {
	a.panel = panel
	return a
}

// Panel devuelve los pedidos ofrecidos y en reparto de la central. Mostrar una
// oferta inicia sus 3 minutos, igual que enviarla por WhatsApp.
func (a *Dispatcher) Panel(ctx context.Context, partner PanelPartner) (domain.PartnerPanel, error) {
	if a.panel == nil {
		return domain.PartnerPanel{}, unavailable("delivery_panel_unavailable", "No pudimos cargar el panel de reparto.", nil)
	}
	fail := func(err error) (domain.PartnerPanel, error) {
		return domain.PartnerPanel{}, unavailable("delivery_panel_unavailable", "No pudimos cargar el panel de reparto.", err)
	}
	if err := a.panel.StartOfferTimers(ctx, partner.ID); err != nil {
		return fail(err)
	}
	orders, err := a.panel.PanelOrders(ctx, partner.ID)
	if err != nil {
		return fail(err)
	}
	failed, err := a.panel.FailedDeliveries(ctx, partner.ID)
	if err != nil {
		return fail(err)
	}
	return domain.PartnerPanel{PartnerName: partner.Name, Active: partner.Active, Orders: orders, FailedDeliveries: failed}, nil
}

// PanelAct aplica una acción del panel: aceptar o rechazar la oferta, liberar
// el pedido antes de recogerlo, mantener al motorizado ante una alerta de
// recogida y registrar la entrega fallida (recoger y entregar siguen aceptándose
// por compatibilidad). Solo vale para ofertas de esa central.
func (a *Dispatcher) PanelAct(ctx context.Context, partner PanelPartner, in domain.PanelActionInput) (domain.ActionResult, error) {
	if !domain.ValidPanelAction(in.Action) {
		return domain.ActionResult{}, failure(apperror.Invalid, "invalid_delivery_action", "La acción no es válida.")
	}
	return a.act(ctx, in.Action, in.Token, func(st OfferState) bool { return st.PartnerID == partner.ID })
}

// DeliveryTracking implementa contracts.TrackingReader: el seguimiento de un
// pedido de delivery. found=false si el pedido no existe o no es delivery.
func (a *Dispatcher) DeliveryTracking(ctx context.Context, s scope, orderID string) (domain.Tracking, bool, error) {
	if a == nil || a.panel == nil || !uuidPattern.MatchString(orderID) {
		return domain.Tracking{}, false, nil
	}
	out, err := a.panel.Tracking(ctx, s, orderID)
	if errors.Is(err, ErrNotFound) {
		return out, false, nil
	}
	return out, err == nil, err
}
