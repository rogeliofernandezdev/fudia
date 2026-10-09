-- A rollback must not discard issued closing reports.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM cash_shift_reports) THEN
    RAISE EXCEPTION 'Cannot remove issued cash shift reports';
  END IF;
END $$;
DROP TABLE cash_shift_reports;
