package auction

// Tournament is a competition container (e.g. league season).
type Tournament struct {
	ID        string
	Name      string
	SportID   string
	StartDate int64 // unix milliseconds
	EndDate   int64 // unix milliseconds
	CreatedAt int64 // unix milliseconds
}
