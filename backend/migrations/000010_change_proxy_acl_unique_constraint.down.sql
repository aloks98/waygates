-- Revert unique constraint from (proxy_id, acl_group_id) back to (proxy_id, path_pattern)

-- Drop the new unique constraint
ALTER TABLE proxy_acl_assignments DROP CONSTRAINT IF EXISTS uq_proxy_acl_assignments_proxy_group;

-- Recreate old unique constraint on (proxy_id, path_pattern)
ALTER TABLE proxy_acl_assignments ADD CONSTRAINT uq_proxy_acl_assignments_proxy_path UNIQUE (proxy_id, path_pattern);
