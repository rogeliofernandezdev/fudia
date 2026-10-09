-- Presentaciones de ingreso para inventario físico.
-- El saldo siempre vive en la unidad base del inventory_item.
CREATE TABLE inventory_presentations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  presentation_type text NOT NULL
    CHECK (presentation_type IN ('unit','package','box')),
  units_per_presentation numeric(14,3) NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  UNIQUE (organization_id, inventory_item_id, presentation_type, units_per_presentation),
  FOREIGN KEY (inventory_item_id, organization_id)
    REFERENCES inventory_items(id, organization_id),
  CONSTRAINT inventory_presentations_factor_check CHECK (
    (presentation_type='unit' AND units_per_presentation=1)
    OR
    (presentation_type IN ('package','box') AND units_per_presentation>1)
  )
);

CREATE INDEX inventory_presentations_item_idx
  ON inventory_presentations(organization_id, inventory_item_id, active);

-- Toda existencia tiene siempre una presentación base 1:1.
INSERT INTO inventory_presentations(
  organization_id, inventory_item_id, presentation_type, units_per_presentation
)
SELECT organization_id, id, 'unit', 1
FROM inventory_items
ON CONFLICT (organization_id, inventory_item_id, presentation_type, units_per_presentation)
DO UPDATE SET active=true, updated_at=now();

-- inventory_entries conserva tanto la presentación recibida como su equivalencia
-- en unidad base. Los registros históricos se consideran entradas por unidad.
ALTER TABLE inventory_entries
  ADD COLUMN presentation_id uuid,
  ADD COLUMN presentation_type text NOT NULL DEFAULT 'unit',
  ADD COLUMN units_per_presentation numeric(14,3) NOT NULL DEFAULT 1,
  ADD COLUMN stock_quantity numeric(14,3)
    GENERATED ALWAYS AS (quantity * units_per_presentation) STORED;

UPDATE inventory_entries ie
SET presentation_id=ip.id
FROM inventory_presentations ip
WHERE ip.organization_id=ie.organization_id
  AND ip.inventory_item_id=ie.inventory_item_id
  AND ip.presentation_type='unit'
  AND ip.units_per_presentation=1;

ALTER TABLE inventory_entries
  ALTER COLUMN presentation_id SET NOT NULL,
  ADD CONSTRAINT inventory_entries_presentation_scope_fk
    FOREIGN KEY (presentation_id, organization_id)
    REFERENCES inventory_presentations(id, organization_id),
  ADD CONSTRAINT inventory_entries_presentation_type_check
    CHECK (presentation_type IN ('unit','package','box')),
  ADD CONSTRAINT inventory_entries_presentation_factor_check CHECK (
    (presentation_type='unit' AND units_per_presentation=1)
    OR
    (presentation_type IN ('package','box') AND units_per_presentation>1)
  );

CREATE INDEX inventory_entries_presentation_idx
  ON inventory_entries(organization_id, presentation_id, created_at DESC);
