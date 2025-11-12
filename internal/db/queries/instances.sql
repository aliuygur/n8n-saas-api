-- name: CreateInstance :one
INSERT INTO instances (
    name, user_id, gke_cluster_name, gke_project_id, gke_zone,
    namespace, domain, worker_replicas, main_cpu_request, main_memory_request,
    worker_cpu_request, worker_memory_request, postgres_storage_size, n8n_storage_size
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetInstance :one
SELECT * FROM instances WHERE id = $1 AND deleted_at IS NULL;

-- name: GetInstanceByName :one
SELECT * FROM instances WHERE name = $1 AND deleted_at IS NULL;

-- name: ListInstancesByUser :many
SELECT * FROM instances 
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListAllInstances :many
SELECT * FROM instances 
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateInstanceStatus :one
UPDATE instances 
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInstanceDeployed :one
UPDATE instances 
SET status = $2, deployed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInstanceResources :one
UPDATE instances 
SET worker_replicas = $2, main_cpu_request = $3, main_memory_request = $4,
    worker_cpu_request = $5, worker_memory_request = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteInstance :one
UPDATE instances 
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteInstance :exec
DELETE FROM instances WHERE id = $1;