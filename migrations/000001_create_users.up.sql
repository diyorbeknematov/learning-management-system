CREATE TYPE user_status AS ENUM (
    'active',
    'blocked'
);

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (id, name)
VALUES
    (gen_random_uuid(), 'SuperAdmin'),
    (gen_random_uuid(), 'Instructor'),
    (gen_random_uuid(), 'Student');

CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    avatar VARCHAR(500),
    bio TEXT,
    role_id UUID NOT NULL REFERENCES roles(id),
    status user_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX users_username_unique_active
ON users (username)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX users_email_unique_active
ON users (email)
WHERE deleted_at IS NULL;

CREATE INDEX refresh_tokens_user_id_idx
ON refresh_tokens (user_id);
