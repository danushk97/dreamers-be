package auction

// WalletTransactionType classifies ledger rows.
type WalletTransactionType string

const (
	WalletTxnCredit WalletTransactionType = "credit"
	WalletTxnDebit  WalletTransactionType = "debit"
)

// Wallet is a team's spendable balance for a tournament/event.
// Amounts are in the smallest currency unit.
type Wallet struct {
	ID           string
	TournamentID string
	EventID      string
	TeamID       string // TournamentTeamRegistration id
	Balance      int64
	CreatedAt    int64 // unix milliseconds
	UpdatedAt    int64 // unix milliseconds
}

// WalletTransaction is an append-only ledger row (settlement, reversal, top-up, etc.).
type WalletTransaction struct {
	ID          string
	WalletID    string
	Amount      int64 // positive magnitude; direction is Type
	Type        WalletTransactionType
	ReferenceID string // e.g. auction_player_id or bid_id depending on integration rules
	CreatedAt   int64 // unix milliseconds
}
