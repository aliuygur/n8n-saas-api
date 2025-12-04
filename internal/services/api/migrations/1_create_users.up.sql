-- Create users table for authentication
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR NOT NULL DEFAULT '',
    name VARCHAR NOT NULL DEFAULT '',
    picture VARCHAR NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);

CREATE UNIQUE INDEX users_email_key ON users(email);
CREATE INDEX idx_users_email ON users(email);

-- Create sessions table for managing user sessions
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR NOT NULL DEFAULT '',
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX sessions_token_key ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
