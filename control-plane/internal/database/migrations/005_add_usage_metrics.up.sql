-- Usage metrics table for tracking API usage per organization
CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    metric VARCHAR(100) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Unique constraint for upsert operations
    CONSTRAINT usage_metrics_org_metric_period_unique UNIQUE (org_id, metric, period_start)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_usage_metrics_org_id ON usage_metrics(org_id);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_metric ON usage_metrics(metric);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_period_start ON usage_metrics(period_start);

