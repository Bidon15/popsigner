-- Remove certificate_pem column
ALTER TABLE client_certificates DROP COLUMN IF EXISTS certificate_pem;

