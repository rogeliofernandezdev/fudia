package postgres

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/jackc/pgx/v5"
)

const channelColumns = `w.id::text,w.country_code::text,c.name,w.phone_number,COALESCE(w.display_name,''),w.active,w.provider,w.revision,w.needs_reconciliation,w.provider_key,w.routing_key`

func scanChannel(row pgx.Row) (domain.PlatformWhatsAppChannel, error) {
	var out domain.PlatformWhatsAppChannel
	err := row.Scan(&out.ID, &out.CountryCode, &out.CountryName, &out.PhoneNumber, &out.DisplayName, &out.Active, &out.Provider, &out.Revision, &out.NeedsReconciliation, &out.ProviderKey, &out.RoutingKey)
	return out, platformError(err)
}

func (r *Platform) whatsAppChannels(ctx context.Context) ([]domain.PlatformWhatsAppChannel, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT `+channelColumns+` FROM platform_whatsapp_channels w JOIN platform_countries c ON c.code=w.country_code WHERE w.superseded_by IS NULL ORDER BY c.name,w.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PlatformWhatsAppChannel{}
	for rows.Next() {
		item, scanErr := scanChannel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Platform) Channel(ctx context.Context, id string) (domain.PlatformWhatsAppChannel, error) {
	return scanChannel(r.db(ctx).QueryRow(ctx, `SELECT `+channelColumns+` FROM platform_whatsapp_channels w JOIN platform_countries c ON c.code=w.country_code WHERE w.id=$1 AND w.superseded_by IS NULL`, id))
}

func (r *Platform) saveWhatsAppChannel(ctx context.Context, s scope, id string, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var out domain.PlatformWhatsAppChannel
	var before any
	if id != "" {
		old, err := r.Channel(ctx, id)
		if err != nil {
			return out, err
		}
		before = old
	}
	if id == "" {
		var createdID string
		err := r.db(ctx).QueryRow(ctx, `INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active,provider,provider_key,routing_key,verified_at)
		 VALUES($1,$2,$3,$4,$5,$6,CASE WHEN $6='' THEN '' ELSE gen_random_uuid()::text END,CASE WHEN $4 THEN now() END) RETURNING id::text`, in.CountryCode, in.PhoneNumber, in.DisplayName, in.Active, in.Provider, in.ProviderKey).Scan(&createdID)
		if err != nil {
			return out, platformError(err)
		}
		id = createdID
	} else {
		tag, err := r.db(ctx).Exec(ctx, `UPDATE platform_whatsapp_channels w SET
		 phone_number=$2,display_name=$3,active=$4,provider=$5,provider_key=$6,
		 routing_key=CASE WHEN $6='' THEN ''
		   WHEN provider=$5 AND provider_key=$6 AND regexp_replace(phone_number,'[^0-9]','','g')=regexp_replace($2,'[^0-9]','','g') AND routing_key<>'' THEN routing_key
		   WHEN routing_key='' AND $5='META_CLOUD_API' AND NOT EXISTS(SELECT 1 FROM platform_whatsapp_routes r WHERE r.routing_key=$6 AND regexp_replace(r.phone_number,'[^0-9]','','g')<>regexp_replace($2,'[^0-9]','','g')) THEN $6
		   ELSE gen_random_uuid()::text END,
		 verified_at=CASE WHEN $4 THEN now() END,needs_reconciliation=false,revision=revision+1,updated_at=now()
		 WHERE id=$1 AND superseded_by IS NULL AND ($7::bigint IS NULL OR revision=$7)`, id, in.PhoneNumber, in.DisplayName, in.Active, in.Provider, in.ProviderKey, in.Revision)
		if err != nil {
			return out, platformError(err)
		}
		if tag.RowsAffected() == 0 {
			return out, platformError(pgx.ErrNoRows)
		}
	}
	out, err := r.Channel(ctx, id)
	if err != nil {
		return out, err
	}
	if out.RoutingKey != "" {
		_, err = r.db(ctx).Exec(ctx, `INSERT INTO platform_whatsapp_routes(routing_key,country_code,provider,provider_key,phone_number)
		 VALUES($1,$2,$3,$4,$5) ON CONFLICT(routing_key) DO NOTHING`, out.RoutingKey, out.CountryCode, out.Provider, out.ProviderKey, out.PhoneNumber)
		if err != nil {
			return out, err
		}
	}
	oldJSON, err := json.Marshal(before)
	if err != nil {
		return out, err
	}
	newJSON, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	_, err = r.db(ctx).Exec(ctx, `INSERT INTO platform_whatsapp_channel_history(channel_id,actor_id,action,before_config,after_config)
	 VALUES($1,NULLIF($2,'')::uuid,$3,$4::jsonb,$5::jsonb)`, id, s.UserID, map[bool]string{true: "channel.created", false: "channel.updated"}[before == nil], oldJSON, newJSON)
	return out, err
}

func (r *Platform) WhatsAppRoutes(ctx context.Context) ([]domain.WhatsAppRoute, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT r.routing_key,r.country_code::text,r.provider,r.provider_key,r.phone_number,
	 EXISTS(SELECT 1 FROM platform_whatsapp_channels w WHERE w.routing_key=r.routing_key AND w.active AND w.superseded_by IS NULL),r.legacy
	 FROM platform_whatsapp_routes r ORDER BY r.created_at,r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.WhatsAppRoute{}
	for rows.Next() {
		var item domain.WhatsAppRoute
		if err = rows.Scan(&item.RoutingKey, &item.CountryCode, &item.Provider, &item.ProviderKey, &item.PhoneNumber, &item.Active, &item.Legacy); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Platform) ChannelHistory(ctx context.Context, id string) ([]map[string]any, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT h.action,h.before_config,h.after_config,h.created_at::text FROM platform_whatsapp_channel_history h
	 JOIN platform_whatsapp_channels w ON w.id=h.channel_id WHERE w.id=$1 OR w.superseded_by=$1 ORDER BY h.created_at DESC,h.id LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var action, at string
		var before, after any
		if err = rows.Scan(&action, &before, &after, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"action": action, "before": before, "after": after, "createdAt": at})
	}
	return out, rows.Err()
}

func (r *Platform) RegisterLegacyRoute(ctx context.Context, in domain.WhatsAppRoute) error {
	_, err := r.db(ctx).Exec(ctx, `WITH route AS (
 INSERT INTO platform_whatsapp_routes(routing_key,country_code,provider,provider_key,phone_number,legacy)
 VALUES($1,$2,$3,$4,$5,true) ON CONFLICT(routing_key) DO NOTHING
 ), restored AS (
 UPDATE platform_whatsapp_channels w SET provider_key=$4,routing_key=$1,active=true,needs_reconciliation=false,verified_at=now(),revision=revision+1,updated_at=now()
 WHERE country_code=$2 AND provider=$3 AND revision=1 AND superseded_by IS NULL AND needs_reconciliation
 AND regexp_replace(phone_number,'[^0-9]','','g')=regexp_replace($5,'[^0-9]','','g')
 AND (SELECT count(*) FROM platform_whatsapp_channel_history h WHERE h.action='migration.imported' AND h.after_config->>'countryCode'=$2 AND (h.after_config->>'active')::boolean)=1
 RETURNING id
 ) INSERT INTO platform_whatsapp_channel_history(channel_id,action,after_config)
 SELECT id,'migration.verified',jsonb_build_object('countryCode',$2::text,'provider',$3::text,'phoneNumber',$5::text,'active',true) FROM restored`, in.RoutingKey, in.CountryCode, in.Provider, in.ProviderKey, in.PhoneNumber)
	return err
}
