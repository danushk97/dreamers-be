-- +goose Up

-- test = sandbox (reset allowed); live = production auction. Distinct from column "mode" (bidding style).
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS run_mode VARCHAR(20) NOT NULL DEFAULT 'test';

ALTER TABLE auctions DROP CONSTRAINT IF EXISTS auctions_run_mode_check;

ALTER TABLE auctions ADD CONSTRAINT auctions_run_mode_check CHECK (run_mode IN ('test', 'live'));

-- +goose Down

ALTER TABLE auctions DROP CONSTRAINT IF EXISTS auctions_run_mode_check;
ALTER TABLE auctions DROP COLUMN IF EXISTS run_mode;
