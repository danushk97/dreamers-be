package gin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/pkg/sanitize"
	playeruc "github.com/dreamers-be/internal/usecase/player"
)

// PlayerHandler handles player HTTP endpoints.
type PlayerHandler struct {
	create *playeruc.CreateUseCase
	list   *playeruc.ListUseCase
}

// NewPlayerHandler returns a new player handler.
func NewPlayerHandler(create *playeruc.CreateUseCase, list *playeruc.ListUseCase) *PlayerHandler {
	return &PlayerHandler{create: create, list: list}
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

	in := &playeruc.CreateInput{
		Name:               sanitize.String(req.Name),
		ImageURL:           sanitize.URL(req.ImageURL),
		AadharCardImageURL: sanitize.URL(req.AadharCardImageURL),
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
		Error(c, http.StatusBadRequest, "Validation Error", err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"player": toPlayerResponse(p)})
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
		Error(c, http.StatusInternalServerError, "Internal Server Error", "failed to list players")
		return
	}

	items := make([]gin.H, len(res.Players))
	for i, p := range res.Players {
		items[i] = toPlayerResponse(p)
	}

	c.JSON(http.StatusOK, gin.H{
		"players": items,
		"total":   res.Total,
	})
}

func toPlayerResponse(p *player.Entity) gin.H {
	phoneNum, _ := strconv.ParseInt(p.Phone, 10, 64)
	return gin.H{
		"id":                 p.ID,
		"name":               p.Name,
		"imageURL":           p.ImageURL,
		"gender":             p.Gender,
		"dateOfBirth":        p.DateOfBirth.Format("2006-01-02"),
		"tnbaId":             p.TNBAID,
		"district":           p.District,
		"phone":              phoneNum,
		"recentAchievements": p.RecentAchievements,
		"tshirtSize":         p.TshirtSize,
		"aadharCardImageURL": p.AadharCardImageURL,
	}
}
