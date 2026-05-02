-- +goose Up
ALTER TABLE wallet_transactions
    ADD COLUMN IF NOT EXISTS wallet_balance BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE wallet_transactions
    DROP COLUMN IF EXISTS wallet_balance;
