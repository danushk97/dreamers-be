-- +goose Up
-- Which lot the auctioneer is showing (console + SSE relay). NULL = use legacy heuristic (is_active / first open lot).
ALTER TABLE auctions
    ADD COLUMN IF NOT EXISTS display_auction_player_id VARCHAR(36) NULL REFERENCES auction_players (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_auctions_display_lot ON auctions (display_auction_player_id);

-- +goose Down
DROP INDEX IF EXISTS idx_auctions_display_lot;
ALTER TABLE auctions DROP COLUMN IF EXISTS display_auction_player_id;
