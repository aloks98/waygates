-- Add OAuth user restriction columns to acl_waygates_auth
ALTER TABLE acl_waygates_auth
ADD COLUMN IF NOT EXISTS allowed_emails TEXT,  -- JSON array of specific emails ["john@example.com"]
ADD COLUMN IF NOT EXISTS allowed_domains TEXT,  -- JSON array of domains ["@company.com"]
ADD COLUMN IF NOT EXISTS allowed_providers TEXT;  -- JSON array of OAuth providers ["google", "github"]

-- Add OAuth user info columns to acl_sessions for OAuth users
ALTER TABLE acl_sessions
ADD COLUMN IF NOT EXISTS email VARCHAR(255),  -- OAuth user's email
ADD COLUMN IF NOT EXISTS provider VARCHAR(50),  -- OAuth provider (google, github, etc.)
ALTER COLUMN user_id DROP NOT NULL;  -- Make user_id nullable for OAuth users

-- Create index for session email lookups
CREATE INDEX IF NOT EXISTS idx_acl_sessions_email ON acl_sessions(email);
CREATE INDEX IF NOT EXISTS idx_acl_sessions_provider ON acl_sessions(provider);
