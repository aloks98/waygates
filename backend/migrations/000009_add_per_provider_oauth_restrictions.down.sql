-- Drop the per-provider OAuth restrictions table
DROP INDEX IF EXISTS idx_acl_oauth_provider_restrictions_group_id;
DROP TABLE IF EXISTS acl_oauth_provider_restrictions;
