-- Eyesite Database Schema
-- PostgreSQL 16+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For full-text search

-- Create schemas for organization
CREATE SCHEMA IF NOT EXISTS main;      -- Users, workspaces, auth
CREATE SCHEMA IF NOT EXISTS spotlight;  -- AI-specific data
CREATE SCHEMA IF NOT EXISTS telemetry;  -- OpenTelemetry data

-- =============================================================================
-- MAIN SCHEMA: Users, Workspaces, Authentication
-- =============================================================================

-- Users table
CREATE TABLE main.users (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth users
    oauth_provider VARCHAR(50),  -- 'google', 'github', NULL for email
    oauth_id VARCHAR(255),
    full_name VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_auth CHECK (
        (password_hash IS NOT NULL AND oauth_provider IS NULL) OR
        (oauth_provider IS NOT NULL AND oauth_id IS NOT NULL)
    )
);

CREATE INDEX idx_users_email ON main.users(email);
CREATE INDEX idx_users_oauth ON main.users(oauth_provider, oauth_id) WHERE oauth_provider IS NOT NULL;

-- Workspaces table (multi-tenancy)
CREATE TABLE main.workspaces (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_uuid UUID NOT NULL REFERENCES main.users(uuid) ON DELETE CASCADE,
    slug VARCHAR(100) UNIQUE NOT NULL, -- URL-friendly identifier
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workspaces_owner ON main.workspaces(owner_uuid);
CREATE INDEX idx_workspaces_slug ON main.workspaces(slug);

-- Workspace members table
CREATE TABLE main.workspace_members (
    workspace_uuid UUID REFERENCES main.workspaces(uuid) ON DELETE CASCADE,
    user_uuid UUID REFERENCES main.users(uuid) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- 'owner', 'admin', 'developer', 'viewer'
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (workspace_uuid, user_uuid),
    CHECK (role IN ('owner', 'admin', 'developer', 'viewer'))
);

CREATE INDEX idx_workspace_members_user ON main.workspace_members(user_uuid);

-- API Keys table
CREATE TABLE main.api_keys (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid UUID NOT NULL REFERENCES main.users(uuid) ON DELETE CASCADE,
    workspace_uuid UUID NOT NULL REFERENCES main.workspaces(uuid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- SHA-256 hex hash of the key (deterministic for ingest lookup)
    key_preview VARCHAR(16) NOT NULL, -- First 8 chars for display
    scopes TEXT[] DEFAULT ARRAY['read', 'write'], -- Permissions
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_api_keys_user ON main.api_keys(user_uuid);
CREATE INDEX idx_api_keys_workspace ON main.api_keys(workspace_uuid);
CREATE INDEX idx_api_keys_hash ON main.api_keys(key_hash);

-- =============================================================================
-- SPOTLIGHT SCHEMA: AI Provider Data
-- =============================================================================

-- Providers table
CREATE TABLE spotlight.providers (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL, -- 'openai', 'anthropic', 'google'
    display_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    pricing JSONB, -- Model-specific pricing {model: {input: x, output: y}}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default providers
INSERT INTO spotlight.providers (name, display_name, pricing) VALUES
('openai', 'OpenAI', '{
    "gpt-4": {"input": 0.00003, "output": 0.00006},
    "gpt-4-turbo": {"input": 0.00001, "output": 0.00003},
    "gpt-3.5-turbo": {"input": 0.0000015, "output": 0.000002}
}'::jsonb),
('anthropic', 'Anthropic', '{
    "claude-3-opus-20240229": {"input": 0.000015, "output": 0.000075},
    "claude-3-sonnet-20240229": {"input": 0.000003, "output": 0.000015},
    "claude-3-haiku-20240307": {"input": 0.00000025, "output": 0.00000125}
}'::jsonb),
('google', 'Google', '{
    "gemini-pro": {"input": 0.00000025, "output": 0.0000005},
    "gemini-pro-vision": {"input": 0.00000025, "output": 0.0000005}
}'::jsonb);

-- Sessions table
CREATE TABLE spotlight.sessions (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid UUID NOT NULL REFERENCES main.users(uuid) ON DELETE CASCADE,
    workspace_uuid UUID NOT NULL REFERENCES main.workspaces(uuid) ON DELETE CASCADE,
    api_key_uuid UUID REFERENCES main.api_keys(uuid) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_workspace ON spotlight.sessions(workspace_uuid);
CREATE INDEX idx_sessions_user ON spotlight.sessions(user_uuid);
CREATE INDEX idx_sessions_created ON spotlight.sessions(created_at DESC);

-- Interactions table (LLM interactions)
CREATE TABLE spotlight.interactions (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_uuid UUID REFERENCES spotlight.sessions(uuid) ON DELETE SET NULL,
    workspace_uuid UUID NOT NULL REFERENCES main.workspaces(uuid) ON DELETE CASCADE,
    trace_id VARCHAR(32), -- OpenTelemetry trace ID
    span_id VARCHAR(16),  -- OpenTelemetry span ID
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(255) NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT,
    prompt_tokens INTEGER DEFAULT 0,
    response_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    cost DECIMAL(12, 8) DEFAULT 0, -- Cost in USD
    latency_ms INTEGER, -- Response time in milliseconds
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'success', 'error'
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (status IN ('pending', 'success', 'error'))
);

CREATE INDEX idx_interactions_workspace ON spotlight.interactions(workspace_uuid);
CREATE INDEX idx_interactions_session ON spotlight.interactions(session_uuid);
CREATE INDEX idx_interactions_trace ON spotlight.interactions(trace_id);
CREATE INDEX idx_interactions_created ON spotlight.interactions(created_at DESC);
CREATE INDEX idx_interactions_provider_model ON spotlight.interactions(provider, model);
CREATE INDEX idx_interactions_cost ON spotlight.interactions(cost DESC);
-- Full-text search index on prompt and response
CREATE INDEX idx_interactions_prompt_trgm ON spotlight.interactions USING gin(prompt gin_trgm_ops);
CREATE INDEX idx_interactions_response_trgm ON spotlight.interactions USING gin(response gin_trgm_ops);

-- =============================================================================
-- TELEMETRY SCHEMA: OpenTelemetry Data
-- =============================================================================

-- Traces table
CREATE TABLE telemetry.traces (
    trace_id VARCHAR(32) PRIMARY KEY,
    user_uuid UUID NOT NULL REFERENCES main.users(uuid) ON DELETE CASCADE,
    workspace_uuid UUID NOT NULL REFERENCES main.workspaces(uuid) ON DELETE CASCADE,
    service_name VARCHAR(255),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    status VARCHAR(20) DEFAULT 'ok', -- 'ok', 'error'
    attributes JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (status IN ('ok', 'error'))
);

CREATE INDEX idx_traces_workspace ON telemetry.traces(workspace_uuid);
CREATE INDEX idx_traces_user ON telemetry.traces(user_uuid);
CREATE INDEX idx_traces_start_time ON telemetry.traces(start_time DESC);
CREATE INDEX idx_traces_status ON telemetry.traces(status);

-- Spans table
CREATE TABLE telemetry.spans (
    span_id VARCHAR(16) PRIMARY KEY,
    trace_id VARCHAR(32) NOT NULL REFERENCES telemetry.traces(trace_id) ON DELETE CASCADE,
    parent_span_id VARCHAR(16),
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(50) DEFAULT 'internal', -- 'internal', 'server', 'client', 'producer', 'consumer'
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    status VARCHAR(20) DEFAULT 'ok',
    attributes JSONB DEFAULT '{}'::jsonb,
    events JSONB DEFAULT '[]'::jsonb, -- Array of span events
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (kind IN ('internal', 'server', 'client', 'producer', 'consumer')),
    CHECK (status IN ('ok', 'error'))
);

CREATE INDEX idx_spans_trace ON telemetry.spans(trace_id);
CREATE INDEX idx_spans_parent ON telemetry.spans(parent_span_id) WHERE parent_span_id IS NOT NULL;
CREATE INDEX idx_spans_start_time ON telemetry.spans(start_time DESC);

-- =============================================================================
-- FUNCTIONS & TRIGGERS
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON main.users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workspaces_updated_at BEFORE UPDATE ON main.workspaces
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON spotlight.sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically add workspace owner as member
CREATE OR REPLACE FUNCTION add_workspace_owner_as_member()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO main.workspace_members (workspace_uuid, user_uuid, role)
    VALUES (NEW.uuid, NEW.owner_uuid, 'owner');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workspace_add_owner_member AFTER INSERT ON main.workspaces
    FOR EACH ROW EXECUTE FUNCTION add_workspace_owner_as_member();

-- =============================================================================
-- VIEWS FOR COMMON QUERIES
-- =============================================================================

-- View for workspace analytics
CREATE OR REPLACE VIEW spotlight.workspace_analytics AS
SELECT 
    w.uuid as workspace_uuid,
    w.name as workspace_name,
    COUNT(DISTINCT i.uuid) as total_interactions,
    SUM(i.total_tokens) as total_tokens,
    SUM(i.cost) as total_cost,
    COUNT(DISTINCT DATE(i.created_at)) as active_days,
    COUNT(DISTINCT i.provider) as providers_used,
    MAX(i.created_at) as last_interaction_at
FROM main.workspaces w
LEFT JOIN spotlight.interactions i ON i.workspace_uuid = w.uuid
GROUP BY w.uuid, w.name;

-- View for user session summary
CREATE OR REPLACE VIEW spotlight.session_summary AS
SELECT
    s.uuid as session_uuid,
    s.name as session_name,
    s.workspace_uuid,
    COUNT(i.uuid) as interaction_count,
    SUM(i.total_tokens) as total_tokens,
    SUM(i.cost) as total_cost,
    MIN(i.created_at) as first_interaction,
    MAX(i.created_at) as last_interaction
FROM spotlight.sessions s
LEFT JOIN spotlight.interactions i ON i.session_uuid = s.uuid
GROUP BY s.uuid, s.name, s.workspace_uuid;

-- Grant permissions (adjust based on your setup)
-- GRANT ALL ON SCHEMA main TO agent_obs_user;
-- GRANT ALL ON SCHEMA spotlight TO agent_obs_user;
-- GRANT ALL ON SCHEMA telemetry TO agent_obs_user;
-- GRANT ALL ON ALL TABLES IN SCHEMA main TO agent_obs_user;
-- GRANT ALL ON ALL TABLES IN SCHEMA spotlight TO agent_obs_user;
-- GRANT ALL ON ALL TABLES IN SCHEMA telemetry TO agent_obs_user;
