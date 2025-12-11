-- Create checkout_sessions table
CREATE TABLE checkout_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    polar_checkout_id VARCHAR NOT NULL UNIQUE,
    subdomain VARCHAR NOT NULL,
    user_email VARCHAR NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'pending',
    success_url VARCHAR NOT NULL,
    return_url VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_checkout_sessions_user_id ON checkout_sessions(user_id);
CREATE INDEX idx_checkout_sessions_polar_checkout_id ON checkout_sessions(polar_checkout_id);
CREATE INDEX idx_checkout_sessions_status ON checkout_sessions(status);

-- Checkout session status can be: 'pending', 'completed', 'expired', 'failed'
