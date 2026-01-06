-- Nitro Infrastructure Registry
-- Tracks deployed RollupCreator contracts per parent chain.
-- This allows reusing infrastructure across multiple customer rollups.

CREATE TABLE IF NOT EXISTS nitro_infrastructure (
    parent_chain_id BIGINT PRIMARY KEY,
    rollup_creator_address TEXT NOT NULL,
    bridge_creator_address TEXT,
    version TEXT NOT NULL,
    deployed_at TIMESTAMPTZ DEFAULT NOW(),
    deployed_by UUID REFERENCES users(id),
    deployment_tx_hash TEXT,
    
    -- Additional deployed infrastructure addresses
    osp_entry_address TEXT,
    challenge_manager_templates_address TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for quick lookups
CREATE INDEX IF NOT EXISTS idx_nitro_infrastructure_version 
    ON nitro_infrastructure(version);

-- Comments for documentation
COMMENT ON TABLE nitro_infrastructure IS 'Tracks deployed Nitro infrastructure (RollupCreator, etc.) per parent chain';
COMMENT ON COLUMN nitro_infrastructure.parent_chain_id IS 'Chain ID of the parent chain (e.g., 1 for mainnet, 11155111 for Sepolia)';
COMMENT ON COLUMN nitro_infrastructure.rollup_creator_address IS 'Address of the deployed RollupCreator contract';
COMMENT ON COLUMN nitro_infrastructure.version IS 'Version of nitro-contracts used (e.g., v3.2.0-beta.0)';
