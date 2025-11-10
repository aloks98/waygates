-- Remove tls_insecure_skip_verify column from proxies table
ALTER TABLE proxies DROP COLUMN tls_insecure_skip_verify;
