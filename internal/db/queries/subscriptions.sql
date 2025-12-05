-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id,
    status,
    trial_started_at,
    trial_ends_at,
    instance_count
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetSubscriptionByUserID :one
SELECT * FROM subscriptions
WHERE user_id = $1
LIMIT 1;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET status = $2,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UpdateSubscriptionPolarInfo :exec
UPDATE subscriptions
SET polar_customer_id = $2,
    polar_subscription_id = $3,
    billing_anchor_date = $4,
    updated_at = NOW()
WHERE user_id = $1;

-- name: IncrementInstanceCount :exec
UPDATE subscriptions
SET instance_count = instance_count + 1,
    updated_at = NOW()
WHERE user_id = $1;

-- name: DecrementInstanceCount :exec
UPDATE subscriptions
SET instance_count = instance_count - 1,
    updated_at = NOW()
WHERE user_id = $1
AND instance_count > 0;

-- name: GetExpiredTrials :many
SELECT * FROM subscriptions
WHERE status = 'trial'
AND trial_ends_at < NOW();

-- name: UpdateSubscriptionToExpired :exec
UPDATE subscriptions
SET status = 'expired',
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateSubscriptionStatusByPolarID :exec
UPDATE subscriptions
SET status = $2,
    updated_at = NOW()
WHERE polar_subscription_id = $1;