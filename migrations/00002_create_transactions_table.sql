-- +goose Up
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    source_account UUID NOT NULL REFERENCES accounts (id),
    destination_account UUID NOT NULL REFERENCES accounts (id),
    amount NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE transactions;
