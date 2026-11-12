package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

// Authenticated PN evidence can link a previously anonymous LID identity to a
// verified telephone. The old visits remain intact under that alias. A proven
// alias already bound to another telephone is never reassigned.
func customerIdentity(ctx context.Context, db database.Executor, namespace string, in domain.CustomerIdentity) (string, error) {
	if !in.Valid() {
		return "", errors.New("invalid identity")
	}
	var id string
	key := in.SubjectKey(namespace)
	if _, err := db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); err != nil {
		return "", err
	}
	if in.VerifiedPhone != "" {
		err := db.QueryRow(ctx, `INSERT INTO concierge_customer_identities(identity_key,verified_phone) VALUES($1,$2)
		 ON CONFLICT(identity_key) DO UPDATE SET identity_key=EXCLUDED.identity_key RETURNING id::text`, in.PhoneKey(), in.VerifiedPhone).Scan(&id)
		if err != nil {
			return "", err
		}
		var alias string
		err = db.QueryRow(ctx, `INSERT INTO concierge_customer_identities(identity_key,canonical_id) VALUES($1,$2)
		 ON CONFLICT(identity_key) DO UPDATE SET canonical_id=EXCLUDED.canonical_id
		 WHERE concierge_customer_identities.verified_phone='' AND (concierge_customer_identities.canonical_id IS NULL OR concierge_customer_identities.canonical_id=EXCLUDED.canonical_id)
		 RETURNING id::text`, key, id).Scan(&alias)
		return id, err
	}
	err := db.QueryRow(ctx, `INSERT INTO concierge_customer_identities(identity_key) VALUES($1)
	 ON CONFLICT(identity_key) DO UPDATE SET identity_key=EXCLUDED.identity_key RETURNING COALESCE(canonical_id,id)::text`, key).Scan(&id)
	return id, err
}

func (r *Usage) RecordConversationOrigin(ctx context.Context, qr application.QRContext, in domain.ConversationInput) error {
	identity, err := customerIdentity(ctx, r.db(ctx), in.ProviderKey, in.Identity)
	if err != nil {
		return err
	}
	var originID string
	err = r.db(ctx).QueryRow(ctx, `WITH inserted AS (
	 INSERT INTO concierge_conversation_origins(conversation_id,organization_id,location_id,routing_key,recipient,identity_id)
	 VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(conversation_id) DO NOTHING RETURNING id
	), visited AS (
	 INSERT INTO concierge_venue_visits(identity_id,organization_id,location_id,customer_id)
	 SELECT $6,$2,$3,(SELECT id FROM customers WHERE organization_id=$2 AND regexp_replace(phone,'[^0-9]','','g')=$7 AND $7<>'' ORDER BY created_at,id LIMIT 1)
	 WHERE EXISTS(SELECT 1 FROM inserted)
	 ON CONFLICT(identity_id,location_id) DO UPDATE SET visit_count=concierge_venue_visits.visit_count+1,last_visited_at=now(),customer_id=COALESCE(concierge_venue_visits.customer_id,EXCLUDED.customer_id)
	)
	 SELECT id::text FROM inserted UNION ALL
	 SELECT id::text FROM concierge_conversation_origins WHERE conversation_id=$1 AND organization_id=$2 AND location_id=$3 AND routing_key=$4 AND recipient=$5 AND identity_id=$6
	 LIMIT 1`, in.ConversationID, qr.Scope.OrganizationID, qr.Scope.LocationID, in.ChannelKey, in.Recipient, identity, in.Identity.VerifiedPhone).Scan(&originID)
	return err
}

const eligibleRestaurants = `FROM locations l
 JOIN organizations o ON o.id=l.organization_id AND o.active
 JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id AND fp.active
 JOIN organization_modules om ON om.organization_id=o.id AND om.module_key='whatsapp_bot' AND om.active
 `

func (r *Concierge) RestaurantDirectory(ctx context.Context, country, namespace string, identity domain.CustomerIdentity, visited bool, offset int) (domain.RestaurantPage, error) {
	out := domain.RestaurantPage{Items: []domain.RestaurantView{}, Offset: offset}
	id, err := customerIdentity(ctx, r.db(ctx), namespace, identity)
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT o.id::text,l.id::text,COALESCE(NULLIF(btrim(o.trade_name),''),o.legal_name),l.name,
	 COALESCE(sum(v.visit_count),0)::bigint,COALESCE(max(v.last_visited_at)::text,'')
	 `+eligibleRestaurants+`
	 LEFT JOIN concierge_venue_visits v ON v.location_id=l.id AND v.identity_id IN(SELECT id FROM concierge_customer_identities WHERE id=$2 OR canonical_id=$2)
	 WHERE l.active AND fp.country_code=$1
	 GROUP BY o.id,l.id HAVING NOT $3::boolean OR count(v.id)>0
	 ORDER BY max(v.last_visited_at) DESC NULLS LAST,o.trade_name,l.name,l.id LIMIT 11 OFFSET $4`, country, id, visited, offset)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.RestaurantView
		if err = rows.Scan(&item.OrganizationID, &item.LocationID, &item.OrganizationName, &item.LocationName, &item.VisitCount, &item.LastVisitedAt); err != nil {
			return out, err
		}
		out.Items = append(out.Items, item)
	}
	if len(out.Items) > 10 {
		out.HasMore = true
		out.Items = out.Items[:10]
	}
	return out, rows.Err()
}

func (r *Concierge) RestaurantContext(ctx context.Context, country, location string) (domain.RestaurantView, error) {
	var out domain.RestaurantView
	err := r.db(ctx).QueryRow(ctx, `WITH eligible AS (
	 SELECT o.id AS organization_id,l.id AS location_id,COALESCE(NULLIF(btrim(o.trade_name),''),o.legal_name) AS organization_name,l.name AS location_name
	 `+eligibleRestaurants+` WHERE l.id=$2 AND l.active AND fp.country_code=$1
	), link AS (
	 INSERT INTO location_delivery_links(organization_id,location_id)
	 SELECT organization_id,location_id FROM eligible
	 ON CONFLICT(location_id) DO UPDATE SET location_id=EXCLUDED.location_id
	 RETURNING location_id,qr_token,active
	)
	 SELECT e.organization_id::text,e.location_id::text,e.organization_name,e.location_name,k.qr_token
	 FROM eligible e JOIN link k ON k.location_id=e.location_id AND k.active`, country, location).Scan(&out.OrganizationID, &out.LocationID, &out.OrganizationName, &out.LocationName, &out.QRToken)
	return out, conciergeError(err)
}
