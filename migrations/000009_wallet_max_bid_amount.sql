-- +goose Up
-- Per-team ceiling for a single bid (paisa). 0 = no team-specific cap (event rules + balance still apply).
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS max_bid_amount BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE wallets DROP COLUMN IF EXISTS max_bid_amount;
