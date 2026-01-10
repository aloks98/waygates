-- Change unique constraint from (proxy_id, path_pattern) to (proxy_id, acl_group_id)
-- This allows multiple ACL groups to be assigned to the same proxy and path pattern (union behavior)

-- Drop the old unique constraint (PostgreSQL requires dropping constraint, not just index)
ALTER TABLE proxy_acl_assignments DROP CONSTRAINT IF EXISTS uq_proxy_acl_assignments_proxy_path;

-- Create new unique constraint on (proxy_id, acl_group_id)
-- Each ACL group can only be assigned once per proxy
ALTER TABLE proxy_acl_assignments ADD CONSTRAINT uq_proxy_acl_assignments_proxy_group UNIQUE (proxy_id, acl_group_id);
