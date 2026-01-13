-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id,
    product_id,
    customer_id,
    subscription_id,
    trial_ends_at,
    status,
    quantity
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSubscriptionByProviderID :one
SELECT * FROM subscriptions
WHERE subscription_id = $1
LIMIT 1;

-- name: GetSubscriptionByUserID :one
SELECT * FROM subscriptions
WHERE user_id = $1
LIMIT 1;

-- name: UpdateSubscriptionStatusByProviderID :exec
UPDATE subscriptions
SET status = $2,
    updated_at = NOW()
WHERE subscription_id = $1;

-- name: DeleteSubscriptionByID :exec
DELETE FROM subscriptions
WHERE id = $1;

-- name: UpdateSubscriptionQuantity :exec
UPDATE subscriptions
SET quantity = $2,
    updated_at = NOW()
WHERE id = $1;