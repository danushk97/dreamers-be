package auction

import "fmt"

// AuctionStatus is the lifecycle of an auction session (DB column auctions.status).
type AuctionStatus string

const (
	AuctionStatusCreated   AuctionStatus = "created"
	AuctionStatusRunning   AuctionStatus = "running"
	AuctionStatusCompleted AuctionStatus = "completed"
)

// ParseAuctionStatus normalizes and validates a client-provided status string.
func ParseAuctionStatus(s string) (AuctionStatus, error) {
	switch AuctionStatus(s) {
	case AuctionStatusCreated, AuctionStatusRunning, AuctionStatusCompleted:
		return AuctionStatus(s), nil
	default:
		return "", fmt.Errorf("status must be \"created\", \"running\", or \"completed\"")
	}
}
