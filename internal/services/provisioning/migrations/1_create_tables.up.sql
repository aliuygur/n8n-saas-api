-- Create instances table to track n8n deployments
CREATE TABLE instances (
    id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    user_id VARCHAR NOT NULL DEFAULT '',
    status VARCHAR NOT NULL DEFAULT 'pending',
    gke_cluster_name VARCHAR NOT NULL DEFAULT '',
    gke_project_id VARCHAR NOT NULL DEFAULT '',
    gke_zone VARCHAR NOT NULL DEFAULT '',
    namespace VARCHAR NOT NULL DEFAULT '',
    domain VARCHAR NOT NULL DEFAULT '',
    worker_replicas INTEGER NOT NULL DEFAULT 1,
    main_cpu_request VARCHAR NOT NULL DEFAULT '500m',
    main_memory_request VARCHAR NOT NULL DEFAULT '512Mi',
    worker_cpu_request VARCHAR NOT NULL DEFAULT '250m',
    worker_memory_request VARCHAR NOT NULL DEFAULT '256Mi',
    postgres_storage_size VARCHAR NOT NULL DEFAULT '10Gi',
    n8n_storage_size VARCHAR NOT NULL DEFAULT '5Gi',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deployed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_instances_user_id ON instances(user_id);
CREATE INDEX idx_instances_status ON instances(status);
CREATE INDEX idx_instances_name ON instances(name);

-- Create deployments table to track deployment history and events
CREATE TABLE deployments (
    id SERIAL PRIMARY KEY,
    instance_id INTEGER NOT NULL REFERENCES instances(id),
    operation VARCHAR NOT NULL DEFAULT '', -- deploy, scale, update, delete
    status VARCHAR NOT NULL DEFAULT 'pending',
    details JSONB NOT NULL DEFAULT '{}',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_deployments_instance_id ON deployments(instance_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_operation ON deployments(operation);