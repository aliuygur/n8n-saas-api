-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id,
    product_id,
    variant_id,
    customer_id,
    subscription_id,
    trial_ends_at,
    status,
    quantity
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
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

-- name: UpdateSubscriptionByUserID :exec
UPDATE subscriptions
SET product_id = $2,
    variant_id = $3,
    customer_id = $4,
    subscription_id = $5,
    status = $6,
    trial_ends_at = $7,
    quantity = $8,
    updated_at = NOW()
WHERE user_id = $1;

-- name: DeleteSubscriptionByID :exec
DELETE FROM subscriptions
WHERE id = $1;

-- name: UpdateSubscriptionQuantity :exec
UPDATE subscriptions
SET quantity = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSubscriptionTrialEndsAt :one
UPDATE subscriptions
SET trial_ends_at = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;