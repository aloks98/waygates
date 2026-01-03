-- Generic key-value settings table for application configuration
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Default 404 settings
INSERT INTO settings (key, value) VALUES
    ('not_found.mode', 'default'),
    ('not_found.redirect_url', '');
