package auctionservice

import (
	"fmt"

	"github.com/dreamers-be/internal/domain/auction"
)

func requireAuctionRunning(a *auction.Auction) error {
	if a == nil {
		return &ValidationError{Err: fmt.Errorf("auction not found")}
	}
	if a.Status != auction.AuctionStatusRunning {
		return &ValidationError{Err: fmt.Errorf("auction must be in running status to perform this action")}
	}
	return nil
}
