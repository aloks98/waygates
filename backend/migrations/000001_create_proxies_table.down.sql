-- Drop indexes first
DROP INDEX IF EXISTS idx_proxies_created_at;
DROP INDEX IF EXISTS idx_proxies_created_by;
DROP INDEX IF EXISTS idx_proxies_is_active;
DROP INDEX IF EXISTS idx_proxies_type;
DROP INDEX IF EXISTS idx_proxies_hostname;

-- Drop proxies table
DROP TABLE IF EXISTS proxies;
