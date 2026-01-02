-- Remove foreign key constraint for proxies.created_by
ALTER TABLE proxies
DROP CONSTRAINT IF EXISTS fk_proxies_created_by;
