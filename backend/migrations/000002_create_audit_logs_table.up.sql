-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- User information
    user_id INTEGER,  -- NULL for system actions or failed login attempts

    -- Action details
    action VARCHAR(100) NOT NULL,  -- e.g., 'create_proxy', 'update_user', 'login_success'
    resource_type VARCHAR(50),     -- 'proxy', 'user', 'settings', etc.
    resource_id INTEGER,            -- ID of the affected resource
    resource_name VARCHAR(255),     -- Name/identifier for easier reference

    -- Additional context (stored as JSON)
    details TEXT,  -- JSON with action-specific details

    -- Request metadata
    ip_address VARCHAR(45),  -- IPv4 or IPv6
    user_agent TEXT,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'success',  -- 'success' or 'failure'
    error_message TEXT,  -- Only populated on failure

    -- Timestamp
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP

    -- Foreign key constraint will be added after users table is created
    -- FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_resource_id ON audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_status ON audit_logs(status);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_user_created ON audit_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_action_created ON audit_logs(action, created_at);
