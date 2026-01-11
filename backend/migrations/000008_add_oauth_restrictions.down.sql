-- Remove OAuth user info indexes from acl_sessions
DROP INDEX IF EXISTS idx_acl_sessions_email;
DROP INDEX IF EXISTS idx_acl_sessions_provider;

-- Remove OAuth user info columns from acl_sessions
ALTER TABLE acl_sessions DROP COLUMN IF EXISTS email;
ALTER TABLE acl_sessions DROP COLUMN IF EXISTS provider;

-- Remove OAuth restriction columns from acl_waygates_auth
ALTER TABLE acl_waygates_auth DROP COLUMN IF EXISTS allowed_emails;
ALTER TABLE acl_waygates_auth DROP COLUMN IF EXISTS allowed_domains;
ALTER TABLE acl_waygates_auth DROP COLUMN IF EXISTS allowed_providers;
