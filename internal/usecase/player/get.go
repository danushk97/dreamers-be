package player

import (
	"context"

	"github.com/dreamers-be/internal/domain/player"
)

// GetUseCase fetches a single player by ID.
type GetUseCase struct {
	repo player.Repository
}

// NewGetUseCase returns a new get player use case.
func NewGetUseCase(repo player.Repository) *GetUseCase {
	return &GetUseCase{repo: repo}
}

// Get returns the player with the given ID, or nil if not found.
func (uc *GetUseCase) Get(ctx context.Context, id string) (*player.Entity, error) {
	return uc.repo.GetByID(ctx, id)
}
