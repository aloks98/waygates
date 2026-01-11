-- Drop ACL tables in reverse order of creation (respecting foreign key dependencies)
DROP TABLE IF EXISTS acl_sessions;
DROP TABLE IF EXISTS acl_branding;
DROP TABLE IF EXISTS proxy_acl_assignments;
DROP TABLE IF EXISTS acl_waygates_auth;
DROP TABLE IF EXISTS acl_external_providers;
DROP TABLE IF EXISTS acl_basic_auth_users;
DROP TABLE IF EXISTS acl_ip_rules;
DROP TABLE IF EXISTS acl_groups;
