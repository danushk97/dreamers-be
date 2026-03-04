package player

import (
	"context"
	"testing"

	"github.com/dreamers-be/internal/domain/player"
)

type mockListRepo struct {
	result *player.ListResult
	err    error
}

func (m *mockListRepo) Create(ctx context.Context, p *player.Entity) error {
	return nil
}

func (m *mockListRepo) ExistsByTNBAID(ctx context.Context, tnbaID string) (bool, error) {
	return false, nil
}

func (m *mockListRepo) GetByID(ctx context.Context, id string) (*player.Entity, error) {
	return nil, nil
}

func (m *mockListRepo) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	return m.result, m.err
}

func TestListUseCase_List(t *testing.T) {
	ctx := context.Background()
	expected := &player.ListResult{
		Players:   []*player.Entity{},
		Total:     0,
		Page:      0,
		Limit:     20,
		PageCount: 0,
	}
	repo := &mockListRepo{result: expected}
	uc := NewListUseCase(repo)

	res, err := uc.List(ctx, &player.ListFilter{Page: 0, Limit: 20})
	if err != nil {
		t.Fatalf("List() err = %v", err)
	}
	if res != expected {
		t.Error("unexpected result")
	}
}

func TestListUseCase_ListNilFilter(t *testing.T) {
	ctx := context.Background()
	expected := &player.ListResult{Total: 5, Limit: 20}
	repo := &mockListRepo{result: expected}
	uc := NewListUseCase(repo)

	res, err := uc.List(ctx, nil)
	if err != nil {
		t.Fatalf("List(nil) err = %v", err)
	}
	if res != expected {
		t.Error("unexpected result")
	}
}
