-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT NOT NULL,
    username TEXT NOT NULL,
    password_hash TEXT,
    google_id TEXT,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    email_verified_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- Enforce uniqueness only on non-deleted records to support soft deletes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active ON users (email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id_active ON users (google_id) WHERE deleted_at IS NULL;

-- Email verification additions
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_user_id ON email_verifications (user_id);
CREATE INDEX IF NOT EXISTS idx_email_verifications_email ON email_verifications (email);
CREATE INDEX IF NOT EXISTS idx_email_verifications_code_hash ON email_verifications (code_hash);

-- Authorization additions
CREATE TABLE IF NOT EXISTS casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(255) NOT NULL,
    v0 VARCHAR(255),
    v1 VARCHAR(255),
    v2 VARCHAR(255),
    v3 VARCHAR(255),
    v4 VARCHAR(255),
    v5 VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_casbin_rule_ptype ON casbin_rule (ptype);
