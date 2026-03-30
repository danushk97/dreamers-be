package playerservice

import (
	"context"

	"github.com/dreamers-be/internal/domain/player"
)


// Get returns the player with the given ID, or nil if not found.
func (uc *PlayerService) Get(ctx context.Context, id string) (*player.Entity, error) {
	return uc.repo.GetByID(ctx, id)
}

