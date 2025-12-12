-- Remove user_id column from api_keys
ALTER TABLE api_keys DROP COLUMN IF EXISTS user_id;

