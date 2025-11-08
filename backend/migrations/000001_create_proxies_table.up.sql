-- Create proxies table
CREATE TABLE IF NOT EXISTS proxies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Common fields
    type VARCHAR(50) NOT NULL,  -- 'reverse_proxy', 'redirect', 'static'
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,

    -- SSL/TLS
    ssl_enabled BOOLEAN NOT NULL DEFAULT 1,
    ssl_forced BOOLEAN NOT NULL DEFAULT 1,

    -- Type-specific configurations (stored as JSON)
    -- For reverse_proxy type
    upstreams TEXT,  -- JSON: [{"host": "...", "port": 8080, "scheme": "http"}]
    load_balancing TEXT,  -- JSON: {"strategy": "round_robin", "health_checks": {...}}
    block_exploits BOOLEAN NOT NULL DEFAULT 1,
    custom_headers TEXT,  -- JSON: {"X-Header": "value"}

    -- For redirect type
    redirect_config TEXT,  -- JSON: {"target": "...", "status_code": 301, ...}

    -- For static type
    static_config TEXT,  -- JSON: {"root_path": "...", "index_file": "index.html", ...}

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT 1,

    -- Metadata
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP

    -- Foreign key constraint will be added after users table is created
    -- FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_proxies_hostname ON proxies(hostname);
CREATE INDEX IF NOT EXISTS idx_proxies_type ON proxies(type);
CREATE INDEX IF NOT EXISTS idx_proxies_is_active ON proxies(is_active);
CREATE INDEX IF NOT EXISTS idx_proxies_created_by ON proxies(created_by);
CREATE INDEX IF NOT EXISTS idx_proxies_created_at ON proxies(created_at);
