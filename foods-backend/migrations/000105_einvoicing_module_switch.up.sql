-- The facturacion module is the only switch: a location issues once its
-- issuer, credentials and series are complete. The per-profile flag is gone.
ALTER TABLE einvoicing_profile_settings DROP COLUMN enabled;
