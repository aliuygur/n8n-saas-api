-- name: CreateDeployment :one
INSERT INTO deployments (instance_id, operation, details)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetDeployment :one
SELECT * FROM deployments WHERE id = $1;

-- name: ListDeploymentsByInstance :many
SELECT * FROM deployments 
WHERE instance_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateDeploymentStatus :one
UPDATE deployments 
SET status = $2, error_message = $3, completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeploymentCompleted :one
UPDATE deployments 
SET status = 'completed', completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeploymentFailed :one
UPDATE deployments 
SET status = 'failed', error_message = $2, completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetActiveDeployments :many
SELECT * FROM deployments 
WHERE status IN ('pending', 'running')
ORDER BY started_at ASC;