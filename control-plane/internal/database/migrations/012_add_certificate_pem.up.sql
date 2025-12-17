-- Add certificate_pem column to store the PEM-encoded client certificate
-- This allows re-downloading the certificate (but not the private key)

ALTER TABLE client_certificates 
ADD COLUMN certificate_pem TEXT;

COMMENT ON COLUMN client_certificates.certificate_pem IS 'PEM-encoded client certificate for re-download. Private key is NOT stored.';

