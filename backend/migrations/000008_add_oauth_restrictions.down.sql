-- Remove OAuth user info columns from acl_sessions
DROP INDEX IF EXISTS idx_acl_sessions_email;
DROP INDEX IF EXISTS idx_acl_sessions_provider;
ALTER TABLE acl_sessions
DROP COLUMN IF EXISTS email,
DROP COLUMN IF EXISTS provider;

-- Remove OAuth restriction columns from acl_waygates_auth
ALTER TABLE acl_waygates_auth
DROP COLUMN IF EXISTS allowed_emails,
DROP COLUMN IF EXISTS allowed_domains,
DROP COLUMN IF EXISTS allowed_providers;
