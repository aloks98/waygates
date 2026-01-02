-- Add foreign key constraint for proxies.created_by
ALTER TABLE proxies
ADD CONSTRAINT fk_proxies_created_by
FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;
