-- name: CreateCheckoutSession :one
INSERT INTO checkout_sessions (
    user_id,
    instance_id,
    polar_checkout_id,
    subdomain,
    user_email,
    success_url,
    return_url,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetCheckoutSessionByPolarID :one
SELECT * FROM checkout_sessions
WHERE polar_checkout_id = $1
LIMIT 1;

-- name: UpdateCheckoutSessionStatus :exec
UPDATE checkout_sessions
SET status = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: UpdateCheckoutSessionCompleted :exec
UPDATE checkout_sessions
SET status = 'completed',
    instance_id = $2,
    updated_at = NOW(),
    completed_at = NOW()
WHERE id = $1 AND completed_at IS NULL;

-- name: GetCheckoutSessionByID :one
SELECT * FROM checkout_sessions
WHERE id = $1
LIMIT 1;

-- name: ListCheckoutSessions :many
SELECT * FROM checkout_sessions
ORDER BY created_at DESC
LIMIT $1;
