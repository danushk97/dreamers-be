package player

import "context"

// ListFilter holds filter criteria for listing players.
type ListFilter struct {
	Name   string // substring search (case-insensitive)
	TNBAID string // substring search (case-insensitive)
	Gender string // exact: MALE, FEMALE, or empty for all
	// Age filter depends on gender:
	// MALE: all, below-30, 31-40, 41-50, 50+
	// FEMALE: all, below-30, above-30
	AgeFilter string
	Page      int
	Limit     int
}

// ListResult holds paginated player list result.
type ListResult struct {
	Players   []*Entity
	Total     int64
	Page      int
	Limit     int
	PageCount int
}

// Repository defines persistence operations for players.
// Interface Segregation: single responsibility for player persistence.
type Repository interface {
	Create(ctx context.Context, p *Entity) error
	List(ctx context.Context, f *ListFilter) (*ListResult, error)
	ExistsByTNBAID(ctx context.Context, tnbaID string) (bool, error)
}
