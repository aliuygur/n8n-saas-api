-- Revert service-agnostic names back to polar-specific names in subscriptions table
ALTER TABLE subscriptions RENAME COLUMN product_id TO polar_product_id;
ALTER TABLE subscriptions RENAME COLUMN customer_id TO polar_customer_id;
ALTER TABLE subscriptions RENAME COLUMN subscription_id TO polar_subscription_id;

-- Revert checkout_sessions field
ALTER TABLE checkout_sessions RENAME COLUMN checkout_id TO polar_checkout_id;

-- Revert index name
DROP INDEX IF EXISTS idx_checkout_sessions_checkout_id;
CREATE INDEX idx_checkout_sessions_polar_checkout_id ON checkout_sessions(polar_checkout_id);
