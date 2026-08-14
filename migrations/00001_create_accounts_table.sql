-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    password_hash TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE accounts;
