-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    polar_product_id VARCHAR NOT NULL DEFAULT '',
    polar_customer_id VARCHAR NOT NULL DEFAULT '',
    polar_subscription_id VARCHAR NOT NULL DEFAULT '',
    seats INTEGER NOT NULL DEFAULT 0,
    status VARCHAR NOT NULL DEFAULT 'trial',
    trial_ends_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- unique index on user_id and polar_product_id
CREATE UNIQUE INDEX idx_subscriptions_user_id_polar_product_id ON subscriptions(user_id, polar_product_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- Subscription status can be: 'trial', 'active', 'expired', 'canceled', 'past_due'