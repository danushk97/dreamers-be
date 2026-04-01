package gin

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/domain/storage"
	playersrv "github.com/dreamers-be/internal/server/player"
	playersvc "github.com/dreamers-be/internal/service/player"
)

// PlayerHandler handles player HTTP endpoints.
type PlayerHandler struct {
	server    *playersrv.PlayerServer
	presigner storage.Presigner // optional, for S3 presigned URLs
}

// NewPlayerHandler returns a new player handler.
func NewPlayerHandler(server *playersrv.PlayerServer, presigner storage.Presigner) *PlayerHandler {
	return &PlayerHandler{server: server, presigner: presigner}
}

// Create creates a new player.
// POST /api/v1/players
func (h *PlayerHandler) Create(c *gin.Context) {
	var req playersrv.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	p, err := h.server.Create(c.Request.Context(), &req)
	if err != nil {
		if playersvc.IsValidationError(err) {
			log.Printf("Create player validation error: %v", err)
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
		} else {
			log.Printf("Create player error: %v", err)
			Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		}
		return
	}
	log.Printf("Player created id=%s tnbaId=%s name=%s", p.ID, p.TNBAID, p.Name)
	c.JSON(http.StatusCreated, gin.H{"player": h.toPlayerResponseWithPresign(c, p)})
}

// List lists players with filters.
// GET /api/v1/players?name=&tnbaId=&gender=&ageFilter=&page=0&limit=20
func (h *PlayerHandler) List(c *gin.Context) {
	q := &playersrv.ListQuery{
		Name:      c.Query("name"),
		TNBAID:    c.Query("tnbaId"),
		Gender:    c.Query("gender"),
		AgeFilter: c.Query("ageFilter"),
		Page:      c.Query("page"),
		Limit:     c.Query("limit"),
	}

	res, err := h.server.List(c.Request.Context(), q)
	if err != nil {
		log.Printf("List players error: %v", err)
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}
	log.Printf("List players total=%d", res.Total)

	items := make([]gin.H, len(res.Players))
	for i, p := range res.Players {
		items[i] = h.toPlayerResponse(c, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"players": items,
		"total":   res.Total,
	})
}

// Get returns a single player by ID with presigned URLs.
// GET /api/v1/players/:id
func (h *PlayerHandler) Get(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	p, err := h.server.Get(c.Request.Context(), id)
	if err != nil {
		log.Printf("Get player id=%s error: %v", id, err)
		if playersvc.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}
	if p == nil {
		log.Printf("Get player id=%s not found", id)
		Error(c, http.StatusNotFound, "Not Found", "player not found")
		return
	}
	log.Printf("Get player id=%s name=%s", p.ID, p.Name)
	c.JSON(http.StatusOK, gin.H{"player": h.toPlayerResponseWithPresign(c, p)})
}

// toPlayerResponse returns a player for list (imageURL and aadharCardImageURL as keys, no presigning).
func (h *PlayerHandler) toPlayerResponse(c *gin.Context, p *player.Entity) gin.H {
	return h.toPlayerResponsePresign(c, p, false)
}

// toPlayerResponseWithPresign returns a player with presigned URLs (for Get, Create).
func (h *PlayerHandler) toPlayerResponseWithPresign(c *gin.Context, p *player.Entity) gin.H {
	return h.toPlayerResponsePresign(c, p, true)
}

func (h *PlayerHandler) toPlayerResponsePresign(c *gin.Context, p *player.Entity, presign bool) gin.H {
	phoneNum, _ := strconv.ParseInt(p.Phone, 10, 64)

	imageURL := p.ImageURL
	aadharURL := p.AadharCardImageURL

	if presign && h.presigner != nil {
		// Presign S3 keys for both profile photo and aadhar.
		if p.ImageURL != "" && !strings.HasPrefix(p.ImageURL, "http://") && !strings.HasPrefix(p.ImageURL, "https://") {
			if u, err := h.presigner.Presign(c.Request.Context(), p.ImageURL, 1*time.Hour); err == nil {
				imageURL = u
			}
		}
		if p.AadharCardImageURL != "" && !strings.HasPrefix(p.AadharCardImageURL, "http://") && !strings.HasPrefix(p.AadharCardImageURL, "https://") {
			if u, err := h.presigner.Presign(c.Request.Context(), p.AadharCardImageURL, 1*time.Hour); err == nil {
				aadharURL = u
			}
		}
	}
	// List: presign=false, keep imageURL and aadharURL as stored keys

	return gin.H{
		"id":                 p.ID,
		"name":               p.Name,
		"imageURL":           imageURL,
		"gender":             p.Gender,
		"dateOfBirth":        p.DateOfBirth.Format("2006-01-02"),
		"tnbaId":             p.TNBAID,
		"district":           p.District,
		"phone":              phoneNum,
		"recentAchievements": p.RecentAchievements,
		"tshirtSize":         p.TshirtSize,
		"aadharCardImageURL": aadharURL,
	}
}
