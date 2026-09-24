UPDATE roles
SET permissions=array_remove(array_remove(array_remove(permissions,'expenses.read'),'expenses.manage'),'expenses.void'),
    updated_at=now()
WHERE system_key IN ('administrator','location_manager','accounting','auditor');

UPDATE roles
SET menu_access=array_remove(menu_access,'costos'),updated_at=now()
WHERE system_key IN ('location_manager','accounting','auditor');

UPDATE organization_modules SET active=false,updated_at=now() WHERE module_key='costos';

UPDATE subscription_plans
SET module_keys=array_remove(module_keys,'costos'),updated_at=now()
WHERE code IN ('legacy','escala');

DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;
