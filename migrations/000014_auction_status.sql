-- +goose Up
ALTER TABLE auctions
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'created'
        CHECK (status IN ('created', 'running', 'completed'));

-- Existing rows: treat as already in progress so current deployments keep working.
UPDATE auctions SET status = 'running' WHERE status = 'created';

-- +goose Down
ALTER TABLE auctions DROP COLUMN IF EXISTS status;
