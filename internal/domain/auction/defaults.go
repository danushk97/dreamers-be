package auction

// Default bid bounds (integer paise, 100 paise = ₹1).
const (
	DefaultMinBidAmountPaisa = int64(150_000) // ₹1500
	DefaultMaxBidAmountPaisa = int64(400_000) // ₹4000
)

// ApplyDefaultAuctionRules fills zero fields with product defaults before validate/persist.
func ApplyDefaultAuctionRules(r AuctionRules) AuctionRules {
	if r.MinBidAmount <= 0 {
		r.MinBidAmount = DefaultMinBidAmountPaisa
	}
	if r.MaxBidAmount <= 0 {
		r.MaxBidAmount = DefaultMaxBidAmountPaisa
	}
	return r
}
