CREATE TABLE IF NOT EXISTS proxy_groups (
  id                       SERIAL PRIMARY KEY,
  name                     VARCHAR(255) NOT NULL,
  description              TEXT,
  base_domain              VARCHAR(255),
  ssl_enabled              BOOLEAN,
  ssl_forced               BOOLEAN,
  tls_insecure_skip_verify BOOLEAN,
  block_exploits           BOOLEAN,
  custom_headers           TEXT,
  created_by               INT NOT NULL,
  created_at               TIMESTAMP NOT NULL DEFAULT now(),
  updated_at               TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT uq_proxy_groups_name UNIQUE (name),
  CONSTRAINT fk_proxy_groups_created_by FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS proxy_group_acl_assignments (
  id             SERIAL PRIMARY KEY,
  proxy_group_id INT NOT NULL,
  acl_group_id   INT NOT NULL,
  path_pattern   VARCHAR(500) NOT NULL DEFAULT '/*',
  priority       INT NOT NULL DEFAULT 0,
  enabled        BOOLEAN NOT NULL DEFAULT true,
  created_at     TIMESTAMP NOT NULL DEFAULT now(),
  updated_at     TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT uq_pgaa_group_acl UNIQUE (proxy_group_id, acl_group_id),
  CONSTRAINT fk_pgaa_proxy_group FOREIGN KEY (proxy_group_id)
    REFERENCES proxy_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_pgaa_acl_group FOREIGN KEY (acl_group_id)
    REFERENCES acl_groups(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pgaa_proxy_group_id ON proxy_group_acl_assignments(proxy_group_id);

ALTER TABLE proxies
  ADD COLUMN IF NOT EXISTS group_id INT NULL,
  ADD COLUMN IF NOT EXISTS hostname_label VARCHAR(63) NULL;

ALTER TABLE proxies
  ADD CONSTRAINT fk_proxies_group_id FOREIGN KEY (group_id)
    REFERENCES proxy_groups(id) ON DELETE RESTRICT;

ALTER TABLE proxies
  ADD CONSTRAINT chk_proxies_label_requires_group
    CHECK (hostname_label IS NULL OR group_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_proxies_group_id ON proxies(group_id);

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              DROP NOT NULL,
  ALTER COLUMN ssl_forced               DROP NOT NULL,
  ALTER COLUMN block_exploits           DROP NOT NULL,
  ALTER COLUMN tls_insecure_skip_verify DROP NOT NULL;

ALTER TABLE proxies
  ALTER COLUMN ssl_enabled              DROP DEFAULT,
  ALTER COLUMN ssl_forced               DROP DEFAULT,
  ALTER COLUMN block_exploits           DROP DEFAULT,
  ALTER COLUMN tls_insecure_skip_verify DROP DEFAULT;
