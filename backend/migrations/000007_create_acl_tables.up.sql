-- Create ACL groups table
CREATE TABLE IF NOT EXISTS acl_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    combination_mode VARCHAR(20) NOT NULL DEFAULT 'any',  -- 'any', 'all', 'ip_bypass'
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_groups_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for acl_groups
CREATE INDEX IF NOT EXISTS idx_acl_groups_name ON acl_groups(name);
CREATE INDEX IF NOT EXISTS idx_acl_groups_created_by ON acl_groups(created_by);
CREATE INDEX IF NOT EXISTS idx_acl_groups_created_at ON acl_groups(created_at);

-- Create ACL IP rules table
CREATE TABLE IF NOT EXISTS acl_ip_rules (
    id SERIAL PRIMARY KEY,
    acl_group_id INTEGER NOT NULL,
    rule_type VARCHAR(20) NOT NULL,  -- 'allow', 'deny', 'bypass'
    cidr VARCHAR(50) NOT NULL,  -- IP or CIDR range
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_ip_rules_acl_group FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE
);

-- Create indexes for acl_ip_rules
CREATE INDEX IF NOT EXISTS idx_acl_ip_rules_acl_group_id ON acl_ip_rules(acl_group_id);
CREATE INDEX IF NOT EXISTS idx_acl_ip_rules_rule_type ON acl_ip_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_acl_ip_rules_priority ON acl_ip_rules(priority);

-- Create ACL basic auth users table
CREATE TABLE IF NOT EXISTS acl_basic_auth_users (
    id SERIAL PRIMARY KEY,
    acl_group_id INTEGER NOT NULL,
    username VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,  -- bcrypt hash
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_basic_auth_users_acl_group FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE,
    CONSTRAINT uq_acl_basic_auth_users_group_username UNIQUE (acl_group_id, username)
);

-- Create indexes for acl_basic_auth_users
CREATE INDEX IF NOT EXISTS idx_acl_basic_auth_users_acl_group_id ON acl_basic_auth_users(acl_group_id);
CREATE INDEX IF NOT EXISTS idx_acl_basic_auth_users_username ON acl_basic_auth_users(username);

-- Create ACL external providers table
CREATE TABLE IF NOT EXISTS acl_external_providers (
    id SERIAL PRIMARY KEY,
    acl_group_id INTEGER NOT NULL,
    provider_type VARCHAR(50) NOT NULL,  -- 'authelia', 'authentik', 'custom'
    name VARCHAR(255) NOT NULL,
    verify_url VARCHAR(500) NOT NULL,
    auth_redirect_url VARCHAR(500),
    headers_to_copy TEXT,  -- JSON array
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_external_providers_acl_group FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE
);

-- Create indexes for acl_external_providers
CREATE INDEX IF NOT EXISTS idx_acl_external_providers_acl_group_id ON acl_external_providers(acl_group_id);
CREATE INDEX IF NOT EXISTS idx_acl_external_providers_provider_type ON acl_external_providers(provider_type);

-- Create ACL Waygates auth table
CREATE TABLE IF NOT EXISTS acl_waygates_auth (
    id SERIAL PRIMARY KEY,
    acl_group_id INTEGER UNIQUE NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    allowed_users TEXT,  -- JSON array of user IDs
    allowed_roles TEXT,  -- JSON array of role keys
    allowed_email_patterns TEXT,  -- JSON array of patterns like '*@company.com'
    require_2fa BOOLEAN DEFAULT false,
    session_ttl INTEGER DEFAULT 86400,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_waygates_auth_acl_group FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE
);

-- Create index for acl_waygates_auth
CREATE INDEX IF NOT EXISTS idx_acl_waygates_auth_acl_group_id ON acl_waygates_auth(acl_group_id);

-- Create proxy ACL assignments table
CREATE TABLE IF NOT EXISTS proxy_acl_assignments (
    id SERIAL PRIMARY KEY,
    proxy_id INTEGER NOT NULL,
    acl_group_id INTEGER NOT NULL,
    path_pattern VARCHAR(500) NOT NULL DEFAULT '/*',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_proxy_acl_assignments_proxy FOREIGN KEY (proxy_id) REFERENCES proxies(id) ON DELETE CASCADE,
    CONSTRAINT fk_proxy_acl_assignments_acl_group FOREIGN KEY (acl_group_id) REFERENCES acl_groups(id) ON DELETE CASCADE,
    CONSTRAINT uq_proxy_acl_assignments_proxy_path UNIQUE (proxy_id, path_pattern)
);

-- Create indexes for proxy_acl_assignments
CREATE INDEX IF NOT EXISTS idx_proxy_acl_assignments_proxy_id ON proxy_acl_assignments(proxy_id);
CREATE INDEX IF NOT EXISTS idx_proxy_acl_assignments_acl_group_id ON proxy_acl_assignments(acl_group_id);
CREATE INDEX IF NOT EXISTS idx_proxy_acl_assignments_priority ON proxy_acl_assignments(priority);
CREATE INDEX IF NOT EXISTS idx_proxy_acl_assignments_enabled ON proxy_acl_assignments(enabled);

-- Create ACL branding table (singleton)
CREATE TABLE IF NOT EXISTS acl_branding (
    id SERIAL PRIMARY KEY,
    logo_url VARCHAR(500),
    primary_color VARCHAR(20) DEFAULT '#3b82f6',
    background_color VARCHAR(20) DEFAULT '#ffffff',
    title VARCHAR(255) DEFAULT 'Login Required',
    subtitle TEXT,
    footer_text TEXT,
    custom_css TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert default branding row
INSERT INTO acl_branding (id, title) VALUES (1, 'Login Required');

-- Create ACL sessions table
CREATE TABLE IF NOT EXISTS acl_sessions (
    id SERIAL PRIMARY KEY,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER NOT NULL,
    proxy_id INTEGER,
    ip_address VARCHAR(50),
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_acl_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_acl_sessions_proxy FOREIGN KEY (proxy_id) REFERENCES proxies(id) ON DELETE CASCADE
);

-- Create indexes for acl_sessions
CREATE INDEX IF NOT EXISTS idx_acl_sessions_session_token ON acl_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_acl_sessions_user_id ON acl_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_acl_sessions_proxy_id ON acl_sessions(proxy_id);
CREATE INDEX IF NOT EXISTS idx_acl_sessions_expires_at ON acl_sessions(expires_at);
