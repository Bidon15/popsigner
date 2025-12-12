-- Revert column sizes (may fail if data is too long)
ALTER TABLE api_keys ALTER COLUMN key_prefix TYPE VARCHAR(20);
ALTER TABLE api_keys ALTER COLUMN key_hash TYPE VARCHAR(255);

