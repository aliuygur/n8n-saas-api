-- Create instances table to track n8n deployments
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL DEFAULT gen_random_uuid(),
    status VARCHAR NOT NULL DEFAULT 'pending',
    gke_cluster_name VARCHAR NOT NULL DEFAULT '',
    gke_project_id VARCHAR NOT NULL DEFAULT '',
    gke_zone VARCHAR NOT NULL DEFAULT '',
    namespace VARCHAR NOT NULL DEFAULT '',
    subdomain VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deployed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_instances_user_id ON instances(user_id);
CREATE INDEX idx_instances_status ON instances(status);
CREATE INDEX idx_instances_namespace ON instances(namespace);

-- Add unique constraints only for non-deleted records
CREATE UNIQUE INDEX instances_namespace_active_key ON instances(namespace) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX instances_subdomain_active_key ON instances(subdomain) WHERE deleted_at IS NULL;