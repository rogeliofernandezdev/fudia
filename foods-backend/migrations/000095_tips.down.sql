-- Do not erase financial history when rolling back an application deployment.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM payment_tips) OR EXISTS(SELECT 1 FROM tip_payouts) THEN
  RAISE EXCEPTION 'Cannot remove tips schema with financial history. Roll back application only.';
 END IF;
END $$;
DROP TABLE tip_payout_allocations,tip_payouts,payment_tips,tip_settings;
ALTER TABLE cash_movements DROP CONSTRAINT cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
 CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment'));
