DELETE FROM subscription_plans p
WHERE p.code IN ('emprende','impulso','escala')
  AND NOT EXISTS(
    SELECT 1 FROM organization_subscriptions s WHERE s.plan_id=p.id
  );
