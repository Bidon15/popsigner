-- Remove org_id from deployments table
-- Drop foreign key constraint if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
               WHERE constraint_name = 'fk_deployments_org_id') THEN
        ALTER TABLE deployments DROP CONSTRAINT fk_deployments_org_id;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_deployments_org_id;
ALTER TABLE deployments DROP COLUMN IF EXISTS org_id;
