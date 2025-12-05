-- Create subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    polar_customer_id VARCHAR DEFAULT '',
    polar_subscription_id VARCHAR DEFAULT '',
    status VARCHAR NOT NULL DEFAULT 'trial',
    trial_started_at TIMESTAMP,
    trial_ends_at TIMESTAMP,
    instance_count INTEGER NOT NULL DEFAULT 0,
    billing_anchor_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX subscriptions_user_id_key ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_trial_ends_at ON subscriptions(trial_ends_at);
CREATE INDEX idx_subscriptions_polar_subscription_id ON subscriptions(polar_subscription_id);

-- Subscription status can be: 'trial', 'active', 'expired', 'canceled', 'past_due'