-- +migrate Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('cron', 'one_time', 'interval')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled', 'deleted')),
    schedule VARCHAR(100),
    timezone VARCHAR(50),
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL DEFAULT 'POST',
    headers JSONB,
    payload JSONB,
    timeout INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    tags TEXT[],
    metadata JSONB,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    run_count BIGINT DEFAULT 0,
    fail_count BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for jobs
CREATE INDEX idx_jobs_tenant_id ON jobs(tenant_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_next_run_at ON jobs(next_run_at) WHERE status = 'active';
CREATE INDEX idx_jobs_tenant_status ON jobs(tenant_id, status);

-- Job executions table
CREATE TABLE IF NOT EXISTS job_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'retrying', 'cancelled', 'timeout')),
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration BIGINT,
    attempt INTEGER DEFAULT 1,
    worker_id VARCHAR(100),
    status_code INTEGER,
    request JSONB,
    response JSONB,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for job_executions
CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);
CREATE INDEX idx_job_executions_tenant_id ON job_executions(tenant_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_scheduled_at ON job_executions(scheduled_at);
CREATE INDEX idx_job_executions_job_scheduled ON job_executions(job_id, scheduled_at DESC);
CREATE INDEX idx_job_executions_pending ON job_executions(scheduled_at) WHERE status = 'pending';

-- Job history table (daily aggregates)
CREATE TABLE IF NOT EXISTS job_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    total_duration BIGINT DEFAULT 0,
    avg_duration DOUBLE PRECISION DEFAULT 0,
    min_duration BIGINT DEFAULT 0,
    max_duration BIGINT DEFAULT 0,
    UNIQUE(job_id, date)
);

-- Indexes for job_histories
CREATE INDEX idx_job_histories_job_id ON job_histories(job_id);
CREATE INDEX idx_job_histories_date ON job_histories(date);
CREATE INDEX idx_job_histories_job_date ON job_histories(job_id, date DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_executions_updated_at
    BEFORE UPDATE ON job_executions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
