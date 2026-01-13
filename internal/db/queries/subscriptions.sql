-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id,
    instance_id,
    product_id,
    customer_id,
    subscription_id,
    trial_ends_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSubscriptionByInstanceID :one
SELECT * FROM subscriptions
WHERE instance_id = $1
LIMIT 1;

-- name: GetSubscriptionByProviderID :one
SELECT * FROM subscriptions
WHERE subscription_id = $1
LIMIT 1;

-- name: GetAllSubscriptionsByUserID :many
SELECT * FROM subscriptions
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetSubscriptionByUserIDAndProductID :one
SELECT * FROM subscriptions
WHERE user_id = $1 AND product_id = $2
LIMIT 1;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET status = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: UpdateSubscriptionProviderInfo :exec
UPDATE subscriptions
SET customer_id = $2,
    subscription_id = $3,
    product_id = $4,
    updated_at = NOW()
WHERE instance_id = $1;

-- name: UpdateSubscriptionToExpired :exec
UPDATE subscriptions
SET status = 'expired',
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSubscriptionStatusByProviderID :exec
UPDATE subscriptions
SET status = $2,
    updated_at = NOW()
WHERE subscription_id = $1;

-- name: DeleteSubscriptionByInstanceID :exec
DELETE FROM subscriptions
WHERE instance_id = $1;