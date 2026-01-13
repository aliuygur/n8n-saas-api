-- Rename polar-specific fields to service-agnostic names in subscriptions table
ALTER TABLE subscriptions RENAME COLUMN polar_product_id TO product_id;
ALTER TABLE subscriptions RENAME COLUMN polar_customer_id TO customer_id;
ALTER TABLE subscriptions RENAME COLUMN polar_subscription_id TO subscription_id;

-- Rename polar-specific field in checkout_sessions table
ALTER TABLE checkout_sessions RENAME COLUMN polar_checkout_id TO checkout_id;

-- Update index name to match new column name
DROP INDEX IF EXISTS idx_checkout_sessions_polar_checkout_id;
CREATE INDEX idx_checkout_sessions_checkout_id ON checkout_sessions(checkout_id);
