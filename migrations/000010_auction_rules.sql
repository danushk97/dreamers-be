-- +goose Up
-- Per-auction bid bounds (paisa), JSON e.g. {"MinBidAmount":400000,"MaxBidAmount":1500000}
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS rules JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE auctions DROP COLUMN IF EXISTS rules;
