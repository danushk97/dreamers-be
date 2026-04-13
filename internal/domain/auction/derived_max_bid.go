package auction

// DerivedMaxBidAmount is the maximum single bid a team can place right now such that:
//   - after paying this bid (if it wins), remaining balance can still cover
//     (remainingMinRosterSlots × minBidReservePaisa), where remainingMinRosterSlots is
//     max(0, minPlayersOnTeam - (soldCount + 1));
//   - then capped by auctionMaxPaisa (if > 0) and teamWalletMaxPaisa (if > 0).
//
// All amounts are in the same smallest unit (paise). Returns 0 if balance cannot cover reserves.
func DerivedMaxBidAmount(balance int64, soldCount int, minPlayersOnTeam int, minBidReservePaisa, auctionMaxPaisa, teamWalletMaxPaisa int64) int64 {
	remaining := 0
	if minPlayersOnTeam > 0 {
		remaining = minPlayersOnTeam - (soldCount + 1)
		if remaining < 0 {
			remaining = 0
		}
	}
	reserve := int64(remaining) * minBidReservePaisa
	if balance < reserve {
		return 0
	}
	d := balance - reserve
	if auctionMaxPaisa > 0 && d > auctionMaxPaisa {
		d = auctionMaxPaisa
	}
	if teamWalletMaxPaisa > 0 && d > teamWalletMaxPaisa {
		d = teamWalletMaxPaisa
	}
	return d
}
