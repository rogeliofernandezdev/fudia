package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/jackc/pgx/v5"
)

// clearCourierSQL deja el reparto sin motorizado: su enlace deja de abrir y se
// reinician las alertas y el reporte de entrega.
const clearCourierSQL = `courier_name=NULL,courier_token=NULL,courier_assigned_at=NULL,
	pickup_alert_ack_level=0,pickup_alerted_at=NULL,delivery_issue_reported_at=NULL,delivery_issue_distance_m=NULL,
	courier_latitude=NULL,courier_longitude=NULL,courier_located_at=NULL`

// pickupWaitSinceSQL es desde cuándo se cuenta la espera de recogida: cocina
// terminó (o la central aceptó después) y, si cambió el motorizado, desde ese cambio.
const pickupWaitSinceSQL = `GREATEST(d.ready_notified_at,COALESCE(d.courier_assigned_at,d.ready_notified_at))`

// pickupLevelSQL traduce la espera a nivel de alerta (1 = 3 min, 2 = 5 min, 3 = 8 min).
var pickupLevelSQL = fmt.Sprintf(`CASE WHEN o.status<>'listo' OR d.status<>'accepted' OR d.ready_notified_at IS NULL THEN 0
	WHEN %[1]s<=now()-interval '%[2]d seconds' THEN 3
	WHEN %[1]s<=now()-interval '%[3]d seconds' THEN 2
	WHEN %[1]s<=now()-interval '%[4]d seconds' THEN 1 ELSE 0 END`,
	pickupWaitSinceSQL, int(domain.PickupCriticalAfter.Seconds()), 300, 180)

// courierDistanceSQL mide en metros (haversine) la distancia del motorizado al
// destino, solo si su ubicación es reciente.
var courierDistanceSQL = fmt.Sprintf(`CASE WHEN d.courier_located_at>now()-interval '%d minutes'
	AND o.delivery_latitude IS NOT NULL AND o.delivery_longitude IS NOT NULL THEN
	round(12742000*asin(sqrt(power(sin(radians(o.delivery_latitude-d.courier_latitude)/2),2)
	  +cos(radians(d.courier_latitude))*cos(radians(o.delivery_latitude))*power(sin(radians(o.delivery_longitude-d.courier_longitude)/2),2))))::int END`,
	domain.CourierLocationMaxAgeMinutes)

// AssignCourier guarda el motorizado. El mismo nombre conserva el enlace; otro
// nombre genera un enlace nuevo (el anterior deja de abrir) y reinicia la espera
// de recogida y la ubicación, sin cambiar la etapa del reparto.
func (r *Repository) AssignCourier(ctx context.Context, s scope, dispatchID, name string) (application.CourierAssignment, error) {
	var out application.CourierAssignment
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE delivery_dispatches SET
		  courier_token=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN courier_token ELSE replace(gen_random_uuid()::text,'-','') END,
		  courier_assigned_at=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN courier_assigned_at ELSE now() END,
		  pickup_alert_ack_level=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN pickup_alert_ack_level ELSE 0 END,
		  pickup_alerted_at=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN pickup_alerted_at END,
		  courier_latitude=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN courier_latitude END,
		  courier_longitude=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN courier_longitude END,
		  courier_located_at=CASE WHEN courier_name IS NOT DISTINCT FROM $3 THEN courier_located_at END,
		  courier_name=$3,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND status IN ('accepted','picked_up')
		RETURNING courier_name,courier_token`, dispatchID, s.OrganizationID, name).Scan(&out.Name, &out.Token)
	return out, err
}

func (r *Repository) ClearCourier(ctx context.Context, s scope, dispatchID string) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE delivery_dispatches SET `+clearCourierSQL+`,updated_at=now()
		WHERE id=$1 AND organization_id=$2`, dispatchID, s.OrganizationID)
	return err
}

// ReleaseOffer retira la aceptación de la central que libera el pedido.
func (r *Repository) ReleaseOffer(ctx context.Context, s scope, offerID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers SET status='withdrawn',released_at=now(),responded_at=now()
		WHERE id=$1 AND organization_id=$2 AND status='accepted'`, offerID, s.OrganizationID)
	return err
}

// KeepPickupAlert registra el nivel de alerta que la central decidió mantener.
func (r *Repository) KeepPickupAlert(ctx context.Context, s scope, dispatchID string) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches d SET pickup_alert_ack_level=GREATEST(d.pickup_alert_ack_level,`+pickupLevelSQL+`),updated_at=now()
		FROM orders o
		WHERE d.id=$1 AND d.organization_id=$2 AND o.id=d.order_id AND o.organization_id=d.organization_id
		  AND `+pickupLevelSQL+`>0`, dispatchID, s.OrganizationID)
	return tag.RowsAffected() > 0, err
}

// ReportDeliveryIssue registra «No pude entregar» con la distancia del motorizado
// al destino en ese momento. first=false si ya estaba reportado.
func (r *Repository) ReportDeliveryIssue(ctx context.Context, s scope, dispatchID string) (bool, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches d SET delivery_issue_reported_at=now(),delivery_issue_distance_m=`+courierDistanceSQL+`,updated_at=now()
		FROM orders o
		WHERE d.id=$1 AND d.organization_id=$2 AND o.id=d.order_id AND o.organization_id=d.organization_id
		  AND d.status='picked_up' AND d.delivery_issue_reported_at IS NULL`, dispatchID, s.OrganizationID)
	return tag.RowsAffected() > 0, err
}

// FailedDeliveryFacts reúne los datos y montos del reparto para la entrega fallida.
func (r *Repository) FailedDeliveryFacts(ctx context.Context, s scope, dispatchID string) (application.FailedDeliveryRecord, error) {
	out := application.FailedDeliveryRecord{DispatchID: dispatchID}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT d.order_id::text,d.accepted_partner_id::text,COALESCE(d.courier_name,''),
		       o.customer_name,o.customer_phone,o.address,d.delivery_issue_reported_at,
		       LEAST(o.paid_amount,o.total)::float8,d.delivery_issue_distance_m,
		       o.delivery_latitude IS NOT NULL AND o.delivery_longitude IS NOT NULL,
		       EXISTS (SELECT 1 FROM customer_notices n WHERE n.order_id=o.id AND n.organization_id=o.organization_id AND n.kind='delivery_contact_request'),
		       (SELECT count(*) FROM delivery_failed_deliveries x
		        WHERE x.partner_id=d.accepted_partner_id AND x.fudia_share_eligible AND x.created_at>now()-interval '30 days')::int
		FROM delivery_dispatches d
		JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		WHERE d.id=$1 AND d.organization_id=$2`, dispatchID, s.OrganizationID).Scan(
		&out.OrderID, &out.PartnerID, &out.CourierName, &out.CustomerName, &out.CustomerPhone, &out.Address, &out.ReportedAt,
		&out.Facts.FoodAmount, &out.Facts.DistanceMeters, &out.Facts.DestinationKnown, &out.Facts.CustomerNotified, &out.Facts.CoveredRecently)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}

func (r *Repository) InsertFailedDelivery(ctx context.Context, s scope, rec application.FailedDeliveryRecord, st domain.FailedDeliverySettlement) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO delivery_failed_deliveries(organization_id,location_id,dispatch_id,order_id,partner_id,courier_name,
		  customer_name,customer_phone,address,food_amount,courier_reimbursement,partner_share,fudia_share,
		  fudia_share_eligible,validation_notes,courier_distance_m,customer_notified,reported_at)
		SELECT d.organization_id,d.location_id,d.id,d.order_id,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		FROM delivery_dispatches d WHERE d.id=$1 AND d.organization_id=$2
		ON CONFLICT (dispatch_id) DO NOTHING`,
		rec.DispatchID, s.OrganizationID, rec.PartnerID, rec.CourierName, rec.CustomerName, rec.CustomerPhone, rec.Address,
		st.FoodAmount, st.CourierReimbursement, st.PartnerShare, st.FudiaShare, st.FudiaShareEligible, st.ValidationNotes,
		rec.Facts.DistanceMeters, rec.Facts.CustomerNotified, rec.ReportedAt)
	return err
}

// CourierOffer devuelve la oferta aceptada del reparto que lleva el motorizado.
func (r *Repository) CourierOffer(ctx context.Context, courierToken string) (string, error) {
	var token string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT f.action_token::text FROM delivery_dispatches d
		JOIN delivery_dispatch_offers f ON f.dispatch_id=d.id AND f.partner_id=d.accepted_partner_id AND f.round=d.round AND f.status='accepted'
		WHERE d.courier_token=$1 AND d.status IN ('accepted','picked_up')`, courierToken).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", application.ErrNotFound
	}
	return token, err
}
