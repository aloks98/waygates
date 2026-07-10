UPDATE proxies SET ssl_enabled              = true  WHERE ssl_enabled IS NULL;
UPDATE proxies SET ssl_forced               = true  WHERE ssl_forced IS NULL;
UPDATE proxies SET block_exploits           = true  WHERE block_exploits IS NULL;
UPDATE proxies SET tls_insecure_skip_verify = false WHERE tls_insecure_skip_verify IS NULL;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              SET DEFAULT true,
  ALTER COLUMN ssl_forced               SET DEFAULT true,
  ALTER COLUMN block_exploits           SET DEFAULT true,
  ALTER COLUMN tls_insecure_skip_verify SET DEFAULT false;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              SET NOT NULL,
  ALTER COLUMN ssl_forced               SET NOT NULL,
  ALTER COLUMN block_exploits           SET NOT NULL,
  ALTER COLUMN tls_insecure_skip_verify SET NOT NULL;

ALTER TABLE proxies DROP CONSTRAINT IF EXISTS chk_proxies_label_requires_group;
ALTER TABLE proxies DROP CONSTRAINT IF EXISTS fk_proxies_group_id;
DROP INDEX IF EXISTS idx_proxies_group_id;
ALTER TABLE proxies DROP COLUMN IF EXISTS hostname_label;
ALTER TABLE proxies DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS proxy_group_acl_assignments;
DROP TABLE IF EXISTS proxy_groups;
