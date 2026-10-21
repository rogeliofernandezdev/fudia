package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// RegisterConciergeCustomer deja prerregistrado en la empresa del QR al comensal
// que pide por WhatsApp y devuelve su id para enlazar el pedido. Reutiliza el
// cliente si su número ya existe (también guardado sin código de país, p. ej.
// «987654321» frente a «51987654321»). Corre dentro de la transacción del pedido.
func (r *Orders) RegisterConciergeCustomer(ctx context.Context, s scope, phone string, name string) (string, error) {
	digits := phoneDigits(phone)
	if len(digits) < 7 {
		return "", nil
	}
	// No hay índice único por teléfono: el lock evita duplicar al mismo número
	// cuando llegan pedidos concurrentes.
	if _, err := r.db(ctx).Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('concierge_customer|'||$1::text||'|'||$2::text,0))
	`, s.OrganizationID, digits); err != nil {
		return "", orderError(err)
	}
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text
		FROM (
		  SELECT id,created_at,regexp_replace(phone,'\D','','g') AS digits
		  FROM customers
		  WHERE organization_id=$1 AND active
		) c
		WHERE c.digits=$2 OR (length(c.digits)>=9 AND right($2,length(c.digits))=c.digits)
		ORDER BY c.created_at,c.id
		LIMIT 1
	`, s.OrganizationID, digits).Scan(&id)
	if err == nil {
		_, err = r.db(ctx).Exec(ctx, `
			UPDATE customers
			SET preferred_channel=CASE WHEN preferred_channel='none' THEN 'whatsapp' ELSE preferred_channel END,
			    updated_at=now()
			WHERE id=$1 AND organization_id=$2
		`, id, s.OrganizationID)
		return id, orderError(err)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", orderError(err)
	}
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		displayName = "Cliente WhatsApp " + digits[len(digits)-4:]
	}
	err = r.db(ctx).QueryRow(ctx, `
		INSERT INTO customers(organization_id,display_name,phone,preferred_channel)
		VALUES($1,$2,'+'||$3,'whatsapp')
		RETURNING id::text
	`, s.OrganizationID, displayName, digits).Scan(&id)
	return id, orderError(err)
}

func phoneDigits(phone string) string {
	var b strings.Builder
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
