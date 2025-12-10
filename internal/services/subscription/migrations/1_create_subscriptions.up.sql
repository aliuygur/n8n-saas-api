-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    polar_product_id VARCHAR NOT NULL DEFAULT '',
    polar_customer_id VARCHAR NOT NULL DEFAULT '',
    polar_subscription_id VARCHAR NOT NULL DEFAULT '',
    status VARCHAR NOT NULL DEFAULT 'trial',
    trial_ends_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- unique index on instance_id (one subscription per instance)
CREATE UNIQUE INDEX idx_subscriptions_instance_id ON subscriptions(instance_id);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- Subscription status can be: 'trial', 'active', 'expired', 'canceled', 'past_due'