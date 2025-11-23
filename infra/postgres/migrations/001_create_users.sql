CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    two_factor_secret VARCHAR(16),
    two_factor_enabled BOOLEAN DEFAULT false,
    two_factor_verified BOOLEAN DEFAULT false,
    two_factor_recovery_codes TEXT
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
