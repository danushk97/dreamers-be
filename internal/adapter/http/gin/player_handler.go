package gin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/domain/storage"
	"github.com/dreamers-be/internal/pkg/sanitize"
	playeruc "github.com/dreamers-be/internal/usecase/player"
)

// PlayerHandler handles player HTTP endpoints.
type PlayerHandler struct {
	create    *playeruc.CreateUseCase
	list      *playeruc.ListUseCase
	get       *playeruc.GetUseCase
	presigner storage.Presigner // optional, for S3 presigned URLs
}

// NewPlayerHandler returns a new player handler.
func NewPlayerHandler(create *playeruc.CreateUseCase, list *playeruc.ListUseCase, get *playeruc.GetUseCase, presigner storage.Presigner) *PlayerHandler {
	return &PlayerHandler{create: create, list: list, get: get, presigner: presigner}
}

// CreateRequest represents the JSON body for player registration.
type CreateRequest struct {
	Name               string `json:"name"`
	ImageURL           string `json:"imageURL"`
	AadharCardImageURL string `json:"aadharCardImageURL"`
	Gender             string `json:"gender"`
	DateOfBirth        string `json:"dateOfBirth"`
	TNBAID             string `json:"tnbaId"`
	District           string `json:"district"`
	Phone              any    `json:"phone"` // number or string from JSON
	RecentAchievements string `json:"recentAchievements"`
	TshirtSize         string `json:"tshirtSize"`
}

// Create creates a new player.
// POST /api/v1/players
func (h *PlayerHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	dob, err := time.Parse("2006-01-02", sanitize.String(req.DateOfBirth))
	if err != nil {
		Error(c, http.StatusBadRequest, "Validation Error", "invalid date of birth (use YYYY-MM-DD)")
		return
	}

	phoneStr := ""
	switch v := req.Phone.(type) {
	case float64:
		phoneStr = strconv.FormatInt(int64(v), 10)
	case string:
		phoneStr = v
	default:
		phoneStr = ""
	}

	// imageURL / aadharCardImageURL: S3 key (uploads/...) or external URL
	imageKey := strings.TrimSpace(req.ImageURL)
	aadharKey := strings.TrimSpace(req.AadharCardImageURL)

	in := &playeruc.CreateInput{
		Name:               sanitize.String(req.Name),
		ImageURL:           imageKey,
		AadharCardImageURL: aadharKey,
		Gender:             sanitize.OneOf(req.Gender, []string{player.GenderMale, player.GenderFemale}),
		DateOfBirth:        dob,
		TNBAID:             sanitize.String(req.TNBAID),
		District:           sanitize.OneOf(req.District, player.TamilNaduDistricts),
		Phone:              sanitize.Phone(phoneStr),
		RecentAchievements: sanitize.MaxLen(sanitize.String(req.RecentAchievements), 300),
		TshirtSize:         sanitize.OneOf(req.TshirtSize, player.ValidTshirtSizes),
	}

	p, err := h.create.Create(c.Request.Context(), in)
	if err != nil {
		if playeruc.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
		} else {
			Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"player": h.toPlayerResponseWithPresign(c, p)})
}

// List lists players with filters.
// GET /api/v1/players?name=&tnbaId=&gender=&ageFilter=&page=0&limit=20
func (h *PlayerHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	f := &player.ListFilter{
		Name:       sanitize.String(c.Query("name")),
		TNBAID:     sanitize.String(c.Query("tnbaId")),
		Gender:     sanitize.OneOf(c.Query("gender"), []string{player.GenderMale, player.GenderFemale}),
		AgeFilter:  sanitize.OneOf(c.Query("ageFilter"), []string{"all", "below-30", "31-40", "41-50", "50+", "above-30"}),
		Page:       page,
		Limit:      limit,
	}

	res, err := h.list.List(c.Request.Context(), f)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}

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
	if id == "" {
		Error(c, http.StatusBadRequest, "Bad Request", "player ID required")
		return
	}

	p, err := h.get.Get(c.Request.Context(), id)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
		return
	}
	if p == nil {
		Error(c, http.StatusNotFound, "Not Found", "player not found")
		return
	}

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
		// Get by ID: include presigned URLs for image and aadhar
		if strings.HasPrefix(p.ImageURL, "profile_photo/") || strings.HasPrefix(p.ImageURL, "uploads/") {
			if u, err := h.presigner.Presign(c.Request.Context(), p.ImageURL, 1*time.Hour); err == nil {
				imageURL = u
			}
		}
		if strings.HasPrefix(p.AadharCardImageURL, "aadhar/") || strings.HasPrefix(p.AadharCardImageURL, "uploads/") {
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
