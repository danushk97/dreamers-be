package gin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/mocks"
	playersrv "github.com/dreamers-be/internal/server/player"
	playersvc "github.com/dreamers-be/internal/service/player"
	"github.com/golang/mock/gomock"
)

func TestPlayerHandler_Create_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	ps := playersrv.NewPlayerServer(playersvc.NewPlayerService(repo))
	ph := NewPlayerHandler(ps, nil)

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

	expected := &player.ListResult{Players: []*player.Entity{}, Total: 0, Page: 0, Limit: 20, PageCount: 0}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().List(gomock.Any(), gomock.Any()).Return(expected, nil).Times(1)
	ps := playersrv.NewPlayerServer(playersvc.NewPlayerService(repo))
	ph := NewPlayerHandler(ps, nil)

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
