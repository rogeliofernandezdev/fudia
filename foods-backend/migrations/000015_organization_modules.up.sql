-- Tabla de módulos activos por organización
CREATE TABLE IF NOT EXISTS organization_modules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    module_key text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, module_key)
);

-- Índice para buscar módulos por organización
CREATE INDEX IF NOT EXISTS organization_modules_organization_idx ON organization_modules(organization_id);

-- Seed: activar todos los módulos para todas las organizaciones existentes
INSERT INTO organization_modules (organization_id, module_key, active)
SELECT o.id, m.key, true
FROM organizations o
CROSS JOIN (VALUES
    ('pos'),
    ('pedidos'),
    ('cocina'),
    ('mesas'),
    ('caja'),
    ('clientes'),
    ('productos'),
    ('compras'),
    ('inventario'),
    ('reportes'),
    ('facturacion'),
    ('reservas'),
    ('personal'),
    ('locales'),
    ('fiscal'),
    ('usuarios'),
    ('integraciones'),
    ('costos'),
    ('combos'),
    ('recetas'),
    ('ofertas'),
    ('delivery'),
    ('delivery_apps'),
    ('repartidores'),
    ('carta_qr'),
    ('kiosco'),
    ('crm'),
    ('puntos'),
    ('call_center'),
    ('logistica'),
    ('bi'),
    ('app_manager'),
    ('whatsapp_bot'),
    ('kardex'),
    ('asistencias')
) AS m(key)
ON CONFLICT (organization_id, module_key) DO NOTHING;
