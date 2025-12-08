-- name: CreateSubscription :one
INSERT INTO subscriptions (
    user_id,
    seats,
    polar_product_id,
    polar_customer_id,
    polar_subscription_id,
    trial_ends_at,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSubscriptionByUserID :one
SELECT * FROM subscriptions
WHERE user_id = $1
LIMIT 1;

-- name: GetSubscriptionByUserIDAndProductID :one
SELECT * FROM subscriptions
WHERE user_id = $1 AND polar_product_id = $2
LIMIT 1;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions
SET status = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: UpdateSubscriptionPolarInfo :exec
UPDATE subscriptions
SET polar_customer_id = $2,
    polar_subscription_id = $3,
    polar_product_id = $4,
    updated_at = NOW()
WHERE user_id = $1;

-- name: UpdateSubscriptionSeats :exec
UPDATE subscriptions
SET seats = $2,
    updated_at = NOW()
WHERE user_id = $1;

-- name: IncrementSubscriptionSeats :exec
UPDATE subscriptions
SET seats = seats + 1,
    updated_at = NOW()
WHERE user_id = $1;

-- name: IncrementSubscriptionSeatsByID :exec
UPDATE subscriptions
SET seats = seats + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: DecrementSubscriptionSeats :exec
UPDATE subscriptions
SET seats = seats - 1,
    updated_at = NOW()
WHERE user_id = $1
AND seats > 0;

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