CREATE INDEX audit_log_availability_history_idx
ON audit_log(organization_id,location_id,entity_id,created_at DESC,id DESC)
WHERE entity_type='product' AND action='product.availability_updated';
