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

-- name: CreateSession :one
INSERT INTO sessions (
    user_id, token, expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetSessionByToken :one
SELECT sqlc.embed(s), sqlc.embed(u)
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token = $1 AND s.expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= NOW();

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: AcquireUserLock :exec
SELECT pg_advisory_lock(hashtext($1));

-- name: ReleaseUserLock :exec
SELECT pg_advisory_unlock(hashtext($1));
