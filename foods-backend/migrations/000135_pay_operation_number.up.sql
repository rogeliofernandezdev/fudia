-- Número bancario opcional; no se deriva de event_key ni de identificadores internos.
ALTER TABLE pay_notifications
  ADD COLUMN operation_number text NOT NULL DEFAULT '',
  ADD CONSTRAINT pay_notifications_operation_number_check
    CHECK (operation_number = '' OR operation_number ~ '^[A-Za-z0-9][A-Za-z0-9-]{0,63}$');
