-- name: CreateUser :one
INSERT INTO users (
    email, name
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UpdateUserLastLogin :one
UPDATE users 
SET last_login_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AcquireLock :exec
SELECT pg_advisory_lock(hashtext($1));

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(hashtext($1));
