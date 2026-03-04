package gin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/player"
	playeruc "github.com/dreamers-be/internal/usecase/player"
)

type mockRepo struct{}

func (m *mockRepo) Create(ctx context.Context, p *player.Entity) error { return nil }
func (m *mockRepo) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	return &player.ListResult{Players: []*player.Entity{}, Total: 0, Page: 0, Limit: 20, PageCount: 0}, nil
}

func (m *mockRepo) ExistsByTNBAID(ctx context.Context, tnbaID string) (bool, error) {
	return false, nil
}

type mockUploader struct{}

func (m *mockUploader) Upload(ctx context.Context, filename string, data []byte, contentType string, folder string) (string, error) {
	return "profile_photo/placeholder/" + filename, nil
}

func TestPlayerHandler_Create_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createUC := playeruc.NewCreateUseCase(&mockRepo{}, &mockUploader{})
	listUC := playeruc.NewListUseCase(&mockRepo{})
	ph := NewPlayerHandler(createUC, listUC, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/players", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	ph.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() status = %d, want 400", w.Code)
	}
}

func TestPlayerHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createUC := playeruc.NewCreateUseCase(&mockRepo{}, &mockUploader{})
	listUC := playeruc.NewListUseCase(&mockRepo{})
	ph := NewPlayerHandler(createUC, listUC, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/players", nil)

	ph.List(c)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %d, want 200", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["players"]; !ok {
		t.Error("response missing 'players' key")
	}
}
