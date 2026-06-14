-- Drop indexes first
DROP INDEX IF EXISTS idx_l4_routes_matcher_type;
DROP INDEX IF EXISTS idx_l4_routes_priority;
DROP INDEX IF EXISTS idx_l4_routes_l4_proxy_id;

DROP INDEX IF EXISTS idx_l4_proxies_created_by;
DROP INDEX IF EXISTS idx_l4_proxies_is_active;
DROP INDEX IF EXISTS idx_l4_proxies_protocol;
DROP INDEX IF EXISTS idx_l4_proxies_name;

-- Drop tables (routes first due to foreign key)
DROP TABLE IF EXISTS l4_routes;
DROP TABLE IF EXISTS l4_proxies;
