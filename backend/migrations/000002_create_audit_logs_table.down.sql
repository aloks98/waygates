-- Drop composite indexes first
DROP INDEX IF EXISTS idx_audit_action_created;
DROP INDEX IF EXISTS idx_audit_resource;
DROP INDEX IF EXISTS idx_audit_user_created;

-- Drop single-column indexes
DROP INDEX IF EXISTS idx_audit_status;
DROP INDEX IF EXISTS idx_audit_resource_id;
DROP INDEX IF EXISTS idx_audit_resource_type;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_created_at;
DROP INDEX IF EXISTS idx_audit_user_id;

-- Drop audit_logs table
DROP TABLE IF EXISTS audit_logs;
