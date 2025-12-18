-- Deployment artifacts table for storing genesis files, rollup configs, etc.
CREATE TABLE IF NOT EXISTS deployment_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    artifact_type VARCHAR(255) NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(deployment_id, artifact_type)
);

CREATE INDEX IF NOT EXISTS idx_deployment_artifacts_deployment_id ON deployment_artifacts(deployment_id);

COMMENT ON TABLE deployment_artifacts IS 'Stores deployment artifacts like genesis files, rollup configs, etc.';

-- Deployment transactions table for tracking on-chain transactions
CREATE TABLE IF NOT EXISTS deployment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    stage VARCHAR(100) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deployment_transactions_deployment_id ON deployment_transactions(deployment_id);
CREATE INDEX IF NOT EXISTS idx_deployment_transactions_tx_hash ON deployment_transactions(tx_hash);

COMMENT ON TABLE deployment_transactions IS 'Tracks on-chain transactions for each deployment stage';

