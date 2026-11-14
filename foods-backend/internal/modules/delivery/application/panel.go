package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// Panel web del proveedor de Delivery Fudia. El enlace privado identifica a la
// central; cada acción vuelve a comprobar que la oferta es suya.

// PanelPartner es la central dueña de un enlace de panel.
type PanelPartner struct {
	ID, Name string
	Active   bool
}

// PanelRepository lee y actualiza lo que la central ve en su panel.
type PanelRepository interface {
	PartnerByPanelToken(ctx context.Context, token string) (PanelPartner, error)
	// StartOfferTimers inicia los 3 minutos de las ofertas que el panel muestra por primera vez.
	StartOfferTimers(ctx context.Context, partnerID string) error
	PanelOrders(ctx context.Context, partnerID string) ([]domain.PanelOrder, error)
	// SetCourierLocation guarda la ubicación en los repartos en curso de la central.
	SetCourierLocation(ctx context.Context, partnerID string, latitude, longitude float64) error
	Tracking(ctx context.Context, s scope, orderID string) (domain.Tracking, error)
}

var errPanelNotFound = failure(apperror.NotFound, "delivery_panel_not_found", "Este enlace del panel ya no es válido. Pide uno nuevo a Fudia.")

// WithPanel conecta las consultas del panel web.
func (a *Dispatcher) WithPanel(panel PanelRepository) *Dispatcher {
	a.panel = panel
	return a
}

func (a *Dispatcher) panelPartner(ctx context.Context, token string) (PanelPartner, error) {
	if a.panel == nil || !domain.ValidPanelToken(token) {
		return PanelPartner{}, errPanelNotFound
	}
	partner, err := a.panel.PartnerByPanelToken(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return partner, errPanelNotFound
	}
	if err != nil {
		return partner, unavailable("delivery_panel_unavailable", "No pudimos cargar el panel de reparto.", err)
	}
	return partner, nil
}

// Panel devuelve los pedidos ofrecidos y en reparto de la central. Mostrar una
// oferta inicia sus 3 minutos, igual que enviarla por WhatsApp.
func (a *Dispatcher) Panel(ctx context.Context, token string) (domain.PartnerPanel, error) {
	partner, err := a.panelPartner(ctx, token)
	if err != nil {
		return domain.PartnerPanel{}, err
	}
	fail := func(err error) (domain.PartnerPanel, error) {
		return domain.PartnerPanel{}, unavailable("delivery_panel_unavailable", "No pudimos cargar el panel de reparto.", err)
	}
	if err = a.panel.StartOfferTimers(ctx, partner.ID); err != nil {
		return fail(err)
	}
	orders, err := a.panel.PanelOrders(ctx, partner.ID)
	if err != nil {
		return fail(err)
	}
	return domain.PartnerPanel{PartnerName: partner.Name, Active: partner.Active, Orders: orders}, nil
}

// PanelAct aplica una acción del panel: aceptar o rechazar la oferta, recoger
// el pedido y confirmar la entrega. Solo vale para ofertas de esa central.
func (a *Dispatcher) PanelAct(ctx context.Context, token string, in domain.PanelActionInput) (domain.ActionResult, error) {
	partner, err := a.panelPartner(ctx, token)
	if err != nil {
		return domain.ActionResult{}, err
	}
	if !domain.ValidPanelAction(in.Action) {
		return domain.ActionResult{}, failure(apperror.Invalid, "invalid_delivery_action", "La acción no es válida.")
	}
	return a.act(ctx, in.Action, in.Token, func(st OfferState) bool { return st.PartnerID == partner.ID })
}

// PanelLocation guarda la ubicación del repartidor en los repartos en curso de
// la central; el cliente la ve en el seguimiento de su pedido.
func (a *Dispatcher) PanelLocation(ctx context.Context, token string, in domain.CourierLocation) error {
	partner, err := a.panelPartner(ctx, token)
	if err != nil {
		return err
	}
	if !domain.ValidCoordinates(in.Latitude, in.Longitude) {
		return failure(apperror.Invalid, "invalid_location", "La ubicación no es válida.")
	}
	if err = a.panel.SetCourierLocation(ctx, partner.ID, in.Latitude, in.Longitude); err != nil {
		return unavailable("delivery_panel_unavailable", "No pudimos guardar la ubicación.", err)
	}
	return nil
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
