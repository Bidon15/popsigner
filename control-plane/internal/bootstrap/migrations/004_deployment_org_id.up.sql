-- Add org_id to deployments table for authorization
-- This addresses CRIT-010: Deployment API has NO organization authorization
ALTER TABLE deployments ADD COLUMN org_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';

-- Remove the default after adding (we only needed it for existing rows)
ALTER TABLE deployments ALTER COLUMN org_id DROP DEFAULT;

-- Add index for filtering deployments by organization (common query pattern)
CREATE INDEX idx_deployments_org_id ON deployments(org_id);

-- Add foreign key to organizations table if it exists
-- Note: Using DO block to make this conditional
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'organizations') THEN
        ALTER TABLE deployments ADD CONSTRAINT fk_deployments_org_id 
            FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE;
    END IF;
END $$;
