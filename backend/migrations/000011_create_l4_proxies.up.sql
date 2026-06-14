-- Create L4 (TCP/UDP) proxies table
CREATE TABLE IF NOT EXISTS l4_proxies (
    id SERIAL PRIMARY KEY,

    -- Basic info
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Network configuration
    listen_port INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL DEFAULT 'tcp',

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT true,

    -- Metadata
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT uq_l4_proxies_port_protocol UNIQUE (listen_port, protocol),
    CONSTRAINT fk_l4_proxies_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT chk_l4_proxies_protocol CHECK (protocol IN ('tcp', 'udp')),
    CONSTRAINT chk_l4_proxies_port_range CHECK (listen_port >= 1 AND listen_port <= 65535)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_l4_proxies_name ON l4_proxies(name);
CREATE INDEX IF NOT EXISTS idx_l4_proxies_protocol ON l4_proxies(protocol);
CREATE INDEX IF NOT EXISTS idx_l4_proxies_is_active ON l4_proxies(is_active);
CREATE INDEX IF NOT EXISTS idx_l4_proxies_created_by ON l4_proxies(created_by);

-- Create L4 routes table
CREATE TABLE IF NOT EXISTS l4_routes (
    id SERIAL PRIMARY KEY,

    -- Parent reference
    l4_proxy_id INTEGER NOT NULL,

    -- Route ordering
    priority INTEGER NOT NULL DEFAULT 0,

    -- Matcher configuration
    matcher_type VARCHAR(50) NOT NULL DEFAULT 'any',
    sni_hostnames TEXT[] DEFAULT '{}',
    allowed_ip_ranges TEXT[] DEFAULT '{}',
    regex_pattern VARCHAR(500),

    -- Upstream configuration (stored as JSON array)
    upstreams JSONB NOT NULL DEFAULT '[]',
    load_balancing_policy VARCHAR(50) NOT NULL DEFAULT 'round_robin',

    -- TLS configuration
    tls_terminate BOOLEAN NOT NULL DEFAULT false,
    tls_passthrough BOOLEAN NOT NULL DEFAULT true,

    -- Proxy protocol (optional)
    proxy_protocol_version VARCHAR(5),

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT fk_l4_routes_l4_proxy FOREIGN KEY (l4_proxy_id) REFERENCES l4_proxies(id) ON DELETE CASCADE,
    CONSTRAINT chk_l4_routes_matcher_type CHECK (matcher_type IN ('any', 'tls', 'ssh', 'postgres', 'http', 'rdp', 'socks5', 'remote_ip', 'regexp')),
    CONSTRAINT chk_l4_routes_lb_policy CHECK (load_balancing_policy IN ('round_robin', 'least_conn', 'random', 'first', 'ip_hash'))
);

-- Create indexes for L4 routes
CREATE INDEX IF NOT EXISTS idx_l4_routes_l4_proxy_id ON l4_routes(l4_proxy_id);
CREATE INDEX IF NOT EXISTS idx_l4_routes_priority ON l4_routes(l4_proxy_id, priority);
CREATE INDEX IF NOT EXISTS idx_l4_routes_matcher_type ON l4_routes(matcher_type);
