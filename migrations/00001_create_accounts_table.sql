-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    forename TEXT NOT NULL,
    surname TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_accounts_active_email
ON accounts (email)
WHERE closed_at IS NULL;

-- +goose Down
DROP TABLE accounts;
