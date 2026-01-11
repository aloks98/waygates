-- Create table for per-provider OAuth restrictions
-- This allows different allowed_emails and allowed_domains per OAuth provider within an ACL group
CREATE TABLE IF NOT EXISTS acl_oauth_provider_restrictions (
    id SERIAL PRIMARY KEY,
    acl_group_id INTEGER NOT NULL,
    provider VARCHAR(50) NOT NULL,  -- e.g., "google", "github", "microsoft", "gitlab"
    allowed_emails TEXT,            -- JSON array of specific emails ["john@example.com"]
    allowed_domains TEXT,           -- JSON array of email domains ["@company.com"]
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_oauth_provider_restrictions_group
        FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE,
    CONSTRAINT uq_acl_oauth_provider_restrictions_group_provider
        UNIQUE (acl_group_id, provider)
);

-- Create index for faster lookups by acl_group_id
CREATE INDEX IF NOT EXISTS idx_acl_oauth_provider_restrictions_group_id
    ON acl_oauth_provider_restrictions(acl_group_id);
