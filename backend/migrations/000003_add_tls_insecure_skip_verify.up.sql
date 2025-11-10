-- Add tls_insecure_skip_verify column to proxies table
ALTER TABLE proxies ADD COLUMN tls_insecure_skip_verify BOOLEAN NOT NULL DEFAULT FALSE;
