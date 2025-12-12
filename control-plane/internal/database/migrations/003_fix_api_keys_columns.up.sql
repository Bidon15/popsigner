-- Fix api_keys column sizes to match actual key format
-- Prefix format: bbr_live_xxxxxxxx (up to 20 chars)
-- Hash format: argon2 hash (much longer)

ALTER TABLE api_keys ALTER COLUMN key_prefix TYPE VARCHAR(50);
ALTER TABLE api_keys ALTER COLUMN key_hash TYPE VARCHAR(500);

