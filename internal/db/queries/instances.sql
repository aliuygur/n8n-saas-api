-- name: CreateInstance :one
INSERT INTO instances (
    user_id, namespace, subdomain, status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetInstance :one
SELECT * FROM instances WHERE id = $1 AND deleted_at IS NULL;

-- name: GetInstanceForUpdate :one
SELECT * FROM instances WHERE id = $1 AND deleted_at IS NULL FOR UPDATE;

-- name: GetInstanceByNamespace :one
SELECT * FROM instances WHERE namespace = $1 AND deleted_at IS NULL;

-- name: GetInstanceBySubdomain :one
SELECT * FROM instances WHERE subdomain = $1 AND deleted_at IS NULL;

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

-- name: UpdateInstanceNamespace :one
UPDATE instances 
SET namespace = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CheckNamespaceExists :one
SELECT EXISTS(SELECT 1 FROM instances WHERE namespace = $1 AND deleted_at IS NULL);

-- name: CheckSubdomainExists :one
SELECT EXISTS(SELECT 1 FROM instances WHERE subdomain = $1 AND deleted_at IS NULL);

-- name: CountActiveInstancesByUserID :one
SELECT COUNT(*) FROM instances WHERE user_id = $1 AND deleted_at IS NULL;

-- name: DeleteInstance :exec
UPDATE instances 
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;
