-- Revert unique constraint from (proxy_id, acl_group_id) back to (proxy_id, path_pattern)

-- Drop the new unique constraint
DROP INDEX IF EXISTS uq_proxy_acl_assignments_proxy_group;

-- Recreate old unique constraint on (proxy_id, path_pattern)
CREATE UNIQUE INDEX uq_proxy_acl_assignments_proxy_path ON proxy_acl_assignments(proxy_id, path_pattern);
