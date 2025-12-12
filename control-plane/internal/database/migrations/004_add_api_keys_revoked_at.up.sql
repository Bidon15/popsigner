-- Add revoked_at column to api_keys if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'api_keys' AND column_name = 'revoked_at'
    ) THEN
        ALTER TABLE api_keys ADD COLUMN revoked_at TIMESTAMPTZ;
    END IF;
END $$;

-- Add last_used_at column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'api_keys' AND column_name = 'last_used_at'
    ) THEN
        ALTER TABLE api_keys ADD COLUMN last_used_at TIMESTAMPTZ;
    END IF;
END $$;

-- Add expires_at column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'api_keys' AND column_name = 'expires_at'
    ) THEN
        ALTER TABLE api_keys ADD COLUMN expires_at TIMESTAMPTZ;
    END IF;
END $$;

-- Add created_at column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'api_keys' AND column_name = 'created_at'
    ) THEN
        ALTER TABLE api_keys ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
    END IF;
END $$;

