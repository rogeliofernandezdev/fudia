package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/jackc/pgx/v5"
)

// BoardOrders lista los deliveries del local: los abiertos y los creados en las
// últimas horas, con su despacho, central, motorizado y espera de recogida.
func (r *Repository) BoardOrders(ctx context.Context, s scope) ([]domain.BoardOrder, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT o.id::text,o.code,o.status,o.customer_name,o.customer_phone,o.address,o.total::text,
		       COALESCE(fp.currency_symbol,''),o.created_at,COALESCE(d.status,''),
		       COALESCE(p.name,''),COALESCE(p.whatsapp_phone,''),COALESCE(d.courier_name,''),
		       CASE WHEN d.status='accepted' AND o.status='listo' AND d.ready_notified_at IS NOT NULL
		            THEN GREATEST(0,floor(EXTRACT(EPOCH FROM now()-`+pickupWaitSinceSQL+`)))::int ELSE 0 END,
		       COALESCE(d.pickup_alert_ack_level,0),d.delivery_issue_reported_at,
		       EXISTS (SELECT 1 FROM delivery_failed_deliveries x WHERE x.dispatch_id=d.id),
		       d.accepted_at,d.picked_up_at,d.delivered_at
		FROM orders o
		JOIN locations l ON l.id=o.location_id AND l.organization_id=o.organization_id
		LEFT JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
		LEFT JOIN delivery_dispatches d ON d.order_id=o.id AND d.organization_id=o.organization_id
		LEFT JOIN delivery_partners p ON p.id=d.accepted_partner_id AND d.status IN ('accepted','picked_up','delivered')
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.channel='delivery'
		  AND ((o.completed_at IS NULL AND o.status<>'cancelado') OR o.created_at>now()-make_interval(hours => $3))
		ORDER BY CASE WHEN o.completed_at IS NULL AND o.status NOT IN ('cancelado','entregado') THEN 0 ELSE 1 END,
		         o.created_at DESC,o.id
		LIMIT 200`, s.OrganizationID, s.LocationID, domain.BoardHistoryHours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.BoardOrder{}
	for rows.Next() {
		var o domain.BoardOrder
		var created time.Time
		var reported, accepted, picked, delivered *time.Time
		var ackLevel int
		if err := rows.Scan(&o.ID, &o.Code, &o.Status, &o.CustomerName, &o.CustomerPhone, &o.Address, &o.Total,
			&o.CurrencySymbol, &created, &o.DispatchStatus, &o.PartnerName, &o.PartnerPhone, &o.CourierName,
			&o.PickupWaitSeconds, &ackLevel, &reported, &o.FailedDelivery, &accepted, &picked, &delivered); err != nil {
			return nil, err
		}
		o.CreatedAt = stamp(created)
		level, alert := domain.PickupAlertLevel(time.Duration(o.PickupWaitSeconds) * time.Second)
		o.PickupAlert, o.PickupAlertAcknowledged = alert, level > 0 && ackLevel >= level
		o.IssueReportedAt = optionalStamp(reported)
		o.AcceptedAt, o.PickedUpAt, o.DeliveredAt = optionalStamp(accepted), optionalStamp(picked), optionalStamp(delivered)
		out = append(out, o)
	}
	return out, rows.Err()
}

// LocationPartners lista las centrales activas que cubren el distrito del local.
func (r *Repository) LocationPartners(ctx context.Context, s scope) (domain.LocationPartners, error) {
	out := domain.LocationPartners{Items: []domain.LocationPartner{}}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE((SELECT ds.enabled FROM organization_delivery_settings ds WHERE ds.organization_id=l.organization_id),false),
		       COALESCE(g.name,'')
		FROM locations l LEFT JOIN geographic_areas g ON g.code=l.district_code
		WHERE l.organization_id=$1 AND l.id=$2`, s.OrganizationID, s.LocationID).Scan(&out.Enabled, &out.DistrictName)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.name,p.whatsapp_phone
		FROM delivery_partners p CROSS JOIN locations l
		WHERE p.active AND l.organization_id=$1 AND l.id=$2 AND `+partnerCoverageSQL+`
		ORDER BY p.priority,p.name`, s.OrganizationID, s.LocationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.LocationPartner
		if err := rows.Scan(&p.Name, &p.WhatsAppPhone); err != nil {
			return out, err
		}
		out.Items = append(out.Items, p)
	}
	return out, rows.Err()
}

// LocationFailedDeliveries lista las entregas fallidas del local de los últimos días.
func (r *Repository) LocationFailedDeliveries(ctx context.Context, s scope) ([]domain.LocationFailedDelivery, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT o.code,p.name,x.customer_name,x.address,x.courier_name,COALESCE(fp.currency_symbol,''),
		       x.food_amount::text,x.courier_reimbursement::text,x.partner_share::text,x.fudia_share::text,
		       x.fudia_share_eligible,x.validation_notes,x.created_at
		FROM delivery_failed_deliveries x
		JOIN orders o ON o.id=x.order_id AND o.organization_id=x.organization_id
		JOIN delivery_partners p ON p.id=x.partner_id
		JOIN locations l ON l.id=x.location_id AND l.organization_id=x.organization_id
		LEFT JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
		WHERE x.organization_id=$1 AND x.location_id=$2 AND x.created_at>now()-make_interval(days => $3)
		ORDER BY x.created_at DESC,x.id`, s.OrganizationID, s.LocationID, domain.FailedHistoryDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.LocationFailedDelivery{}
	for rows.Next() {
		var f domain.LocationFailedDelivery
		var registered time.Time
		if err := rows.Scan(&f.OrderCode, &f.PartnerName, &f.CustomerName, &f.Address, &f.CourierName, &f.CurrencySymbol,
			&f.FoodAmount, &f.CourierReimbursement, &f.PartnerShare, &f.FudiaShare,
			&f.FudiaShareEligible, &f.ValidationNotes, &registered); err != nil {
			return nil, err
		}
		if f.ValidationNotes == nil {
			f.ValidationNotes = []string{}
		}
		f.RegisteredAt = stamp(registered)
		out = append(out, f)
	}
	return out, rows.Err()
}
