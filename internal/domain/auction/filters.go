package auction

// RegistrationFilter is a runtime category filter chosen by the auctioneer.
// Zero values mean "no filter" for that field.
type RegistrationFilter struct {
	MinAgeYears int
	MaxAgeYears int
	Gender      string
}

