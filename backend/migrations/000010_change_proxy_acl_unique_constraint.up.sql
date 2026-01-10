-- Change unique constraint from (proxy_id, path_pattern) to (proxy_id, acl_group_id)
-- This allows multiple ACL groups to be assigned to the same proxy and path pattern (union behavior)

-- Drop the old unique constraint
DROP INDEX IF EXISTS uq_proxy_acl_assignments_proxy_path;

-- Create new unique constraint on (proxy_id, acl_group_id)
-- Each ACL group can only be assigned once per proxy
CREATE UNIQUE INDEX uq_proxy_acl_assignments_proxy_group ON proxy_acl_assignments(proxy_id, acl_group_id);
