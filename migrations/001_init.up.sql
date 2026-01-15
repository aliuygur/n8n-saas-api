-- Enable UUID v7 extension (sortable UUIDs)
CREATE EXTENSION IF NOT EXISTS pg_uuidv7;

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create instances table to track n8n deployments
CREATE TABLE instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'pending',
    namespace VARCHAR NOT NULL DEFAULT '',
    subdomain VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deployed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_instances_user_id ON instances(user_id);
CREATE INDEX idx_instances_status ON instances(status);
CREATE INDEX idx_instances_namespace ON instances(namespace);

-- Add unique constraints only for non-deleted records
CREATE UNIQUE INDEX instances_namespace_active_key ON instances(namespace) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX instances_subdomain_active_key ON instances(subdomain) WHERE deleted_at IS NULL;

-- Create subscriptions table (one subscription per user with quantity field for instance limits)
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL,
    product_id VARCHAR NOT NULL DEFAULT '',
    customer_id VARCHAR NOT NULL DEFAULT '',
    subscription_id VARCHAR NOT NULL DEFAULT '',
    status VARCHAR NOT NULL DEFAULT 'trial',
    quantity INTEGER NOT NULL DEFAULT 1,  -- Number of instances allowed
    trial_ends_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for subscriptions
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
-- Ensure one active subscription per user per product (prevents duplicate active subscriptions)
CREATE UNIQUE INDEX idx_subscriptions_user_product ON subscriptions(user_id, product_id) WHERE status NOT IN ('cancelled', 'expired');

-- Subscription status can be: 'trial', 'active', 'expired', 'canceled', 'past_due'

-- Create checkout_sessions table
CREATE TABLE checkout_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL,
    instance_id UUID NOT NULL UNIQUE DEFAULT uuid_generate_v7(),
    subdomain VARCHAR NOT NULL,
    user_email VARCHAR NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'pending',
    success_url VARCHAR NOT NULL,
    return_url VARCHAR NOT NULL,
    checkout_id VARCHAR NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_checkout_sessions_user_id ON checkout_sessions(user_id);
CREATE INDEX idx_checkout_sessions_checkout_id ON checkout_sessions(checkout_id);
CREATE INDEX idx_checkout_sessions_status ON checkout_sessions(status);

-- Checkout session status can be: 'pending', 'completed', 'expired', 'failed'
