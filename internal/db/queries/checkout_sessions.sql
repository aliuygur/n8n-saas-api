-- name: CreateCheckoutSession :one
INSERT INTO checkout_sessions (
    user_id,
    polar_checkout_id,
    subdomain,
    user_email,
    success_url,
    return_url,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetCheckoutSessionByPolarID :one
SELECT * FROM checkout_sessions
WHERE polar_checkout_id = $1
LIMIT 1;

-- name: UpdateCheckoutSessionStatus :exec
UPDATE checkout_sessions
SET status = $2,
    updated_at = NOW(),
    completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
WHERE id = $1;

-- name: GetCheckoutSessionByID :one
SELECT * FROM checkout_sessions
WHERE id = $1
LIMIT 1;
