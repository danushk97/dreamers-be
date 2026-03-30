package playerservice

import (
	"context"

	"github.com/dreamers-be/internal/domain/player"
)


// List returns players matching the filter.
func (uc *PlayerService) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	if f == nil {
		f = &player.ListFilter{Limit: 20}
	}
	return uc.repo.List(ctx, f)
}

