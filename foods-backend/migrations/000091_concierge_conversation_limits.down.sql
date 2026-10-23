DROP TABLE concierge_token_usage;
DROP TABLE concierge_conversations;
DROP TABLE concierge_monthly_usage;
DROP TABLE organization_concierge_limits;
DROP TABLE concierge_usage_settings;
ALTER TABLE subscription_plans DROP COLUMN max_concierge_conversations;
