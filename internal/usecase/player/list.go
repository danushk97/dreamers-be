package player

import (
	"context"

	"github.com/dreamers-be/internal/domain/player"
)

// ListUseCase handles listing players with filters.
type ListUseCase struct {
	repo player.Repository
}

// NewListUseCase returns a new list players use case.
func NewListUseCase(repo player.Repository) *ListUseCase {
	return &ListUseCase{repo: repo}
}

// List returns players matching the filter.
func (uc *ListUseCase) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	if f == nil {
		f = &player.ListFilter{Limit: 20}
	}
	return uc.repo.List(ctx, f)
}
