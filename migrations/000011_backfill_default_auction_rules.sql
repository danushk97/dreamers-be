-- +goose Up
-- Default ₹1500 / ₹4000 min-max single bid (paise) for auctions missing rules.
UPDATE auctions
SET rules = '{"MinBidAmount": 150000, "MaxBidAmount": 400000}'::jsonb
WHERE rules IS NULL OR rules = '{}'::jsonb;

-- +goose Down
-- Cannot restore previous empty rules; no-op.
SELECT 1;
