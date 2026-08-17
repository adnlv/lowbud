-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    forename TEXT NOT NULL,
    surname TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE accounts;
