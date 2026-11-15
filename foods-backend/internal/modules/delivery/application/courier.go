package application

import (
	"context"
	"errors"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// Motorizado de la central, liberación del pedido, alertas de recogida y
// entrega fallida. Todo usa los estados existentes del despacho; los hechos del
// reparto se guardan en sus columnas.

// CourierAssignment es el motorizado vigente y su enlace.
type CourierAssignment struct{ Name, Token string }

// PickupDelay identifica el aviso crítico al restaurante (uno por motorizado).
type PickupDelay struct{ PartnerName, Key string }

// FailedDeliveryRecord reúne lo que se conserva de una entrega fallida.
type FailedDeliveryRecord struct {
	DispatchID, OrderID, PartnerID       string
	CourierName                          string
	CustomerName, CustomerPhone, Address string
	ReportedAt                           time.Time
	Facts                                domain.FailedDeliveryFacts
}

var (
	errCourierNotFound = failure(apperror.NotFound, "delivery_courier_not_found", "Este enlace ya no es válido. Pide uno nuevo a tu central.")
	errCourierName     = failure(apperror.Invalid, "invalid_courier_name", "Escribe el nombre del motorizado (2 a 60 caracteres).")
)

// assignedOutcome valida que la oferta sea la asignación vigente de la central.
// Devuelve "" si el reparto sigue en manos de esa central.
func assignedOutcome(st OfferState) string {
	switch {
	case st.DispatchStatus == domain.DispatchCancelled:
		return domain.OutcomeCancelled
	case st.OfferStatus != domain.OfferAccepted || !st.CurrentRound || !st.Assigned:
		return domain.OutcomeTaken
	case st.DispatchStatus == domain.DispatchDelivered:
		return domain.OutcomeDelivered
	case st.DispatchStatus != domain.DispatchAccepted && st.DispatchStatus != domain.DispatchPickedUp:
		return domain.OutcomeTaken
	}
	return ""
}

// PanelCourier asigna o cambia el motorizado de un reparto aceptado. El mismo
// nombre conserva el enlace (doble clic); otro nombre invalida el anterior y
// genera uno nuevo sin cambiar la etapa del reparto.
func (a *Dispatcher) PanelCourier(ctx context.Context, partner PanelPartner, in domain.CourierInput) (domain.CourierResult, error) {
	name, ok := domain.NormalizeCourierName(in.CourierName)
	if !ok {
		return domain.CourierResult{}, errCourierName
	}
	if !uuidPattern.MatchString(in.Token) {
		return domain.CourierResult{Outcome: domain.OutcomeNotFound}, nil
	}
	fail := func(err error) (domain.CourierResult, error) {
		return domain.CourierResult{}, unavailable("delivery_courier_unavailable", "No pudimos generar el enlace del motorizado.", err)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	tctx := tx.Context()
	st, err := a.repo.LockOfferByToken(tctx, in.Token)
	if errors.Is(err, ErrNotFound) || (err == nil && st.PartnerID != partner.ID) {
		return domain.CourierResult{Outcome: domain.OutcomeNotFound}, nil
	}
	if err != nil {
		return fail(err)
	}
	if outcome := assignedOutcome(st); outcome != "" {
		return domain.CourierResult{Outcome: outcome}, nil
	}
	s := scope{OrganizationID: st.OrganizationID, LocationID: st.LocationID}
	assigned, err := a.repo.AssignCourier(tctx, s, st.DispatchID, name)
	if err != nil {
		return fail(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fail(err)
	}
	return domain.CourierResult{Outcome: domain.OutcomeCourierAssigned, CourierName: assigned.Name, CourierToken: assigned.Token}, nil
}

// release libera un pedido aceptado que aún no se recoge: el enlace del
// motorizado deja de funcionar y el pedido vuelve a ofrecerse a las demás
// centrales en una vuelta nueva.
func (a *Dispatcher) release(ctx context.Context, s scope, st OfferState) (string, *noticecontracts.Notice, error) {
	if outcome := assignedOutcome(st); outcome != "" {
		return outcome, nil, nil
	}
	if st.DispatchStatus == domain.DispatchPickedUp {
		return domain.OutcomeAlreadyPickedUp, nil, nil
	}
	// Con pagos registrados no se mueve el envío del pedido: coordinar con el local.
	if st.HasPayments {
		return domain.OutcomePaymentConflict, nil, nil
	}
	if err := a.repo.ReleaseOffer(ctx, s, st.OfferID); err != nil {
		return "", nil, err
	}
	facts, err := a.repo.OrderFacts(ctx, s, st.OrderID)
	if err != nil {
		return "", nil, err
	}
	d, found, err := a.repo.LockDispatch(ctx, s, st.OrderID)
	if err != nil || !found {
		return "", nil, err
	}
	d.Round++
	if err = a.repo.StartRound(ctx, s, d.ID, d.Round); err != nil {
		return "", nil, err
	}
	d.Status = domain.DispatchSearching
	// El envío vuelve al pedido hasta que otra central acepte.
	if err = a.settleFee(ctx, s, st.OrderID); err != nil {
		return "", nil, err
	}
	if facts.Status == "cancelado" {
		return domain.OutcomeReleased, nil, a.stop(ctx, s, d, domain.DispatchCancelled)
	}
	if !domain.OrderDispatchable(facts.Channel, facts.Status, facts.Completed) {
		return domain.OutcomeReleased, nil, a.stop(ctx, s, d, domain.DispatchOwn)
	}
	notice, err := a.offerNext(ctx, s, facts, d, true)
	return domain.OutcomeReleased, notice, err
}

// keep registra que la central mantiene al motorizado en el nivel de alerta
// actual. No cambia el reparto ni lo libera.
func (a *Dispatcher) keep(ctx context.Context, s scope, st OfferState) (string, error) {
	if outcome := assignedOutcome(st); outcome != "" {
		return outcome, nil
	}
	if st.DispatchStatus == domain.DispatchPickedUp {
		return domain.OutcomeAlreadyPickedUp, nil
	}
	if _, err := a.repo.KeepPickupAlert(ctx, s, st.DispatchID); err != nil {
		return "", err
	}
	return domain.OutcomeKept, nil
}

// reportIssue registra «No pude entregar» del motorizado. true indica que hay
// que pedir al cliente, por su chat, que se comunique.
func (a *Dispatcher) reportIssue(ctx context.Context, s scope, st OfferState) (string, bool, error) {
	if outcome := assignedOutcome(st); outcome != "" {
		return outcome, false, nil
	}
	if st.DispatchStatus != domain.DispatchPickedUp {
		return domain.OutcomeNotPickedUp, false, nil
	}
	switch st.OrderStatus {
	case "entregado":
		return domain.OutcomeDelivered, false, nil
	case "en_camino":
	default:
		return domain.OutcomeNotPickedUp, false, nil
	}
	first, err := a.repo.ReportDeliveryIssue(ctx, s, st.DispatchID)
	return domain.OutcomeIssueReported, first, err
}

// fail registra la entrega fallida: exige el reporte del motorizado y la espera
// mínima para que el cliente responda. Conserva datos y montos para el reclamo,
// cierra el pedido para el restaurante (ya cobró la comida) y cierra el reparto.
func (a *Dispatcher) fail(ctx context.Context, s scope, st OfferState) (string, error) {
	if outcome := assignedOutcome(st); outcome != "" {
		return outcome, nil
	}
	switch {
	case st.DispatchStatus != domain.DispatchPickedUp:
		return domain.OutcomeNotPickedUp, nil
	case st.OrderStatus == "entregado":
		return domain.OutcomeDelivered, nil
	case st.OrderStatus != "en_camino":
		return domain.OutcomeNotPickedUp, nil
	case !st.IssueReported:
		return domain.OutcomeNotReported, nil
	case !st.FailReady:
		return domain.OutcomeTooSoon, nil
	}
	if a.handoff == nil {
		return "", errors.New("order handoff is not configured")
	}
	record, err := a.repo.FailedDeliveryFacts(ctx, s, st.DispatchID)
	if err != nil {
		return "", err
	}
	if err = a.repo.InsertFailedDelivery(ctx, s, record, domain.SettleFailedDelivery(record.Facts)); err != nil {
		return "", err
	}
	if err = a.handoff.MarkDelivered(ctx, s, st.OrderID); err != nil {
		return "", err
	}
	if err = a.repo.ClearCourier(ctx, s, st.DispatchID); err != nil {
		return "", err
	}
	if err = a.repo.SetDispatchStatus(ctx, s, st.DispatchID, domain.DispatchCancelled); err != nil {
		return "", err
	}
	return domain.OutcomeDeliveryFailed, nil
}

// CourierView devuelve el pedido que lleva el motorizado de este enlace.
func (a *Dispatcher) CourierView(ctx context.Context, token string) (domain.CourierView, error) {
	if a.panel == nil || !domain.ValidCourierToken(token) {
		return domain.CourierView{}, errCourierNotFound
	}
	out, err := a.panel.CourierView(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return out, errCourierNotFound
	}
	if err != nil {
		return out, unavailable("delivery_courier_unavailable", "No pudimos cargar el pedido.", err)
	}
	return out, nil
}

// CourierAct aplica una acción del motorizado: recoger, entregar o «No pude
// entregar». Comparte bloqueos y resultados con el panel y los botones.
func (a *Dispatcher) CourierAct(ctx context.Context, token, action string) (domain.ActionResult, error) {
	if !domain.ValidCourierToken(token) {
		return domain.ActionResult{}, errCourierNotFound
	}
	if !domain.ValidCourierAction(action) {
		return domain.ActionResult{}, failure(apperror.Invalid, "invalid_delivery_action", "La acción no es válida.")
	}
	offer, err := a.repo.CourierOffer(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return domain.ActionResult{}, errCourierNotFound
	}
	if err != nil {
		return domain.ActionResult{}, unavailable("delivery_action_unavailable", "No pudimos registrar la acción.", err)
	}
	return a.act(ctx, action, offer, func(st OfferState) bool { return st.CourierToken != "" && st.CourierToken == token })
}

// CourierLocation guarda la ubicación del motorizado en su reparto en curso.
func (a *Dispatcher) CourierLocation(ctx context.Context, token string, in domain.CourierLocation) error {
	if a.panel == nil || !domain.ValidCourierToken(token) {
		return errCourierNotFound
	}
	if !domain.ValidCoordinates(in.Latitude, in.Longitude) {
		return failure(apperror.Invalid, "invalid_location", "La ubicación no es válida.")
	}
	ok, err := a.panel.SetCourierLocationByToken(ctx, token, in.Latitude, in.Longitude)
	if err != nil {
		return unavailable("delivery_courier_unavailable", "No pudimos guardar la ubicación.", err)
	}
	if !ok {
		return errCourierNotFound
	}
	return nil
}
