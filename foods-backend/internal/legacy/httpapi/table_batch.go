package httpapi

import (
	"context"
	"encoding/json"
)

type tableBatchItem struct {
	Name      string `json:"name"`
	Seats     int    `json:"seats"`
	Zone      string `json:"zone"`
	QrEnabled bool   `json:"qrEnabled"`
}

// One statement validates/locks the zones, inserts the entire batch and audits it.
// Its implicit transaction commits before pgx returns the result; there is no
// per-table round trip or separate audit operation after persistence.
const tableBatchSQL = `
	WITH input AS MATERIALIZED (
		SELECT item->>'name' AS name, (item->>'seats')::integer AS seats,
		       item->>'zone' AS zone, (item->>'qrEnabled')::boolean AS qr_enabled,
		       position
		FROM jsonb_array_elements($3::jsonb) WITH ORDINALITY AS entries(item,position)
	), locked_zones AS MATERIALIZED (
		SELECT name FROM zones
		WHERE organization_id=$1 AND location_id=$2 AND active
		  AND name IN (SELECT zone FROM input WHERE zone<>'')
		ORDER BY name FOR SHARE
	), invalid_zones AS MATERIALIZED (
		SELECT zone FROM input
		WHERE zone<>'' AND NOT EXISTS (SELECT 1 FROM locked_zones WHERE name=input.zone)
	), created AS (
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		SELECT $1,$2,name,seats,zone,true,encode(gen_random_bytes(16),'hex'),qr_enabled
		FROM input WHERE NOT EXISTS (SELECT 1 FROM invalid_zones)
		ORDER BY name
		RETURNING id,name,seats,zone,active,qr_token,qr_enabled
	), audited AS (
		INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata)
		SELECT $1::uuid,$2::uuid,$4::uuid,'table.batch_created','table',NULL,
		       jsonb_build_object('tableIds',jsonb_agg(id),'count',count(*))
		FROM created HAVING count(*)>0
		RETURNING id
	)
	SELECT EXISTS(SELECT 1 FROM invalid_zones),
	       jsonb_build_object('items',COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'id',c.id,'name',c.name,'seats',c.seats,'zone',c.zone,'active',c.active,
			'qrToken',c.qr_token,'qrEnabled',c.qr_enabled
		) ORDER BY i.position)
		FROM created c JOIN input i ON i.name=c.name
	       ),'[]'::jsonb))
`

func persistTableBatch(ctx context.Context, q tableRowQuerier, s scope, payload []byte) (json.RawMessage, bool, error) {
	var result json.RawMessage
	var invalidZone bool
	err := q.QueryRow(ctx, tableBatchSQL, s.OrganizationID, s.LocationID, string(payload), s.UserID).Scan(&invalidZone, &result)
	return result, invalidZone, err
}
