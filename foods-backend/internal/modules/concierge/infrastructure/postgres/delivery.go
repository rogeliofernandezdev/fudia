package postgres

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	deliverycontracts "github.com/foods-platform/foods-backend/internal/modules/delivery/contracts"
)

func (r *Concierge) feeTiers(ctx context.Context, s scope) ([]deliverycontracts.FeeTier, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT max_distance_km::float8,fee::float8
		FROM delivery_fee_tiers
		WHERE organization_id=$1 AND location_id=$2
		ORDER BY max_distance_km
	`, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tiers := []deliverycontracts.FeeTier{}
	for rows.Next() {
		var tier deliverycontracts.FeeTier
		if err := rows.Scan(&tier.MaxDistanceKm, &tier.Fee); err != nil {
			return nil, err
		}
		tiers = append(tiers, tier)
	}
	return tiers, rows.Err()
}

func (r *Concierge) DeliverySetup(ctx context.Context, s scope) (application.DeliverySetup, error) {
	var out application.DeliverySetup
	err := r.db(ctx).QueryRow(ctx, `
		SELECT l.address,l.phone,l.latitude::float8,l.longitude::float8
		FROM locations l
		WHERE l.id=$2 AND l.organization_id=$1 AND l.active
	`, s.OrganizationID, s.LocationID).Scan(&out.Address, &out.Phone, &out.Latitude, &out.Longitude)
	if err != nil {
		return out, conciergeError(err)
	}
	out.Tiers, err = r.feeTiers(ctx, s)
	return out, err
}

func (r *Concierge) DeliveryFees(ctx context.Context, s scope) (domain.DeliveryFeesView, []deliverycontracts.FeeTier, error) {
	var out domain.DeliveryFeesView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT l.name,l.latitude IS NOT NULL AND l.longitude IS NOT NULL,COALESCE(p.currency_symbol,'')
		FROM locations l
		LEFT JOIN organization_fiscal_profiles p
		  ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id
		WHERE l.id=$2 AND l.organization_id=$1 AND l.active
	`, s.OrganizationID, s.LocationID).Scan(&out.LocationName, &out.HasCoordinates, &out.CurrencySymbol)
	if err != nil {
		return out, nil, conciergeError(err)
	}
	tiers, err := r.feeTiers(ctx, s)
	return out, tiers, err
}

// ReplaceDeliveryFees reemplaza todos los rangos del local en una transacción.
func (r *Concierge) ReplaceDeliveryFees(ctx context.Context, s scope, tiers []deliverycontracts.FeeTier) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `
		DELETE FROM delivery_fee_tiers WHERE organization_id=$1 AND location_id=$2
	`, s.OrganizationID, s.LocationID); err != nil {
		return err
	}
	for _, tier := range tiers {
		if _, err = tx.Exec(ctx, `
			INSERT INTO delivery_fee_tiers(organization_id,location_id,max_distance_km,fee)
			VALUES($1,$2,$3,$4)
		`, s.OrganizationID, s.LocationID, tier.MaxDistanceKm, tier.Fee); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
