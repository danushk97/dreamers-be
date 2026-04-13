package gin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dreamers-be/internal/domain/auction"
	auctionservice "github.com/dreamers-be/internal/service/auction"
)

type AuctionHandler struct {
	auctionSvc     *auctionservice.AuctionService
	lotSvc         *auctionservice.LotService
	bidSvc         *auctionservice.BidService
	settlementSvc  *auctionservice.SettlementService

	auctionPlayerRepo auction.AuctionPlayerRepository
	bidRepo           auction.BidRepository
	walletRepo       auction.WalletRepository
}

func NewAuctionHandler(
	auctionSvc *auctionservice.AuctionService,
	lotSvc *auctionservice.LotService,
	bidSvc *auctionservice.BidService,
	settlementSvc *auctionservice.SettlementService,
	auctionPlayerRepo auction.AuctionPlayerRepository,
	bidRepo auction.BidRepository,
	walletRepo auction.WalletRepository,
) *AuctionHandler {
	return &AuctionHandler{
		auctionSvc:         auctionSvc,
		lotSvc:             lotSvc,
		bidSvc:             bidSvc,
		settlementSvc:      settlementSvc,
		auctionPlayerRepo: auctionPlayerRepo,
		bidRepo:           bidRepo,
		walletRepo:       walletRepo,
	}
}

// --- Request DTOs ---

type RegistrationFilterRequest struct {
	MinAgeYears int    `json:"minAgeYears"`
	MaxAgeYears int    `json:"maxAgeYears"`
	Gender      string `json:"gender"`
}

func (r RegistrationFilterRequest) toDomain() auction.RegistrationFilter {
	return auction.RegistrationFilter{
		MinAgeYears: r.MinAgeYears,
		MaxAgeYears: r.MaxAgeYears,
		Gender:      r.Gender,
	}
}

type FilterPresetRequest struct {
	ID    string                   `json:"id"`
	Name  string                   `json:"name"`
	Filter RegistrationFilterRequest `json:"filter"`
}

func (r FilterPresetRequest) toDomain(nowMs int64) auction.FilterPreset {
	return auction.FilterPreset{
		ID:        r.ID,
		Name:      r.Name,
		Filter:    r.Filter.toDomain(),
		CreatedAt: nowMs,
	}
}

// AuctionRulesRequest matches auctions.rules JSON keys (MinBidAmount, MaxBidAmount), paise.
type AuctionRulesRequest struct {
	MinBidAmount int64 `json:"MinBidAmount"`
	MaxBidAmount int64 `json:"MaxBidAmount"`
}

type CreateAuctionRequest struct {
	TournamentID      string                `json:"tournamentId"`
	TournamentEventID string                `json:"tournamentEventId"`
	Mode              string                `json:"mode"`
	// RunMode is "test" (default, allows POST .../reset) or "live".
	RunMode           string                `json:"runMode"`
	FilterPresets     []FilterPresetRequest `json:"filterPresets"`
	Rules             AuctionRulesRequest   `json:"rules"`
}

type ListEligibleRequest struct {
	// PresetID: optional — use a saved filter from the auction (from filterPresets).
	// If set, it overrides the inline filter body for this request.
	PresetID string `json:"presetId"`
	Filter   RegistrationFilterRequest `json:"filter"`
}

type CreateLotsBulkRequest struct {
	// Optional: if provided, server will create lots for exactly these registrations.
	RegistrationIDs []string `json:"registrationIds"`

	// When RegistrationIDs is empty, server will query eligible registrations
	// for (tournamentId, tournamentEventId) and create lots automatically.
	TournamentID      string                   `json:"tournamentId"`
	TournamentEventID string                   `json:"tournamentEventId"`
	// PresetID: optional — same semantics as POST .../eligible (saved category on auction).
	PresetID string                   `json:"presetId"`
	Filter   RegistrationFilterRequest `json:"filter"`

	StartLotNumber int   `json:"startLotNumber"`
	BasePrice      int64 `json:"basePrice"` // 0 = use auction rules MinBidAmount
	Limit           int   `json:"limit"`     // 0 = no limit
}

type PlaceBidRequest struct {
	TeamRegistrationID string `json:"teamRegistrationId"`
	Amount             int64  `json:"amount"`
}

type SellRequest struct {
	ExpectedBidID string `json:"expectedBidId"`
}

type RevertSaleRequest struct {
	Reason string `json:"reason"`
}

// --- Response DTOs ---

type AuctionResponse struct {
	ID                string                   `json:"id"`
	TournamentID      string                   `json:"tournamentId"`
	TournamentEventID string                   `json:"tournamentEventId"`
	Mode              auction.AuctionMode     `json:"mode"`
	FilterPresets    []auction.FilterPreset  `json:"filterPresets"`
	CreatedAt         int64                    `json:"createdAt"`
}

type AuctionPlayerResponse struct {
	ID                             string `json:"id"`
	AuctionID                      string `json:"auctionId"`
	TournamentPlayerRegistrationID string `json:"tournamentPlayerRegistrationId"`
	Status                           auction.AuctionPlayerStatus `json:"status"`
	BasePrice                        int64  `json:"basePrice"`
	FinalPrice                       int64  `json:"finalPrice"`
	SoldToTeamRegistrationID         string `json:"soldToTeamRegistrationId"`
	LotNumber                        int    `json:"lotNumber"`
	IsActive                         bool   `json:"isActive"`
	CreatedAt                        int64  `json:"createdAt"`
}

func toAuctionPlayerResponse(ap *auction.AuctionPlayer) AuctionPlayerResponse {
	if ap == nil {
		return AuctionPlayerResponse{}
	}
	return AuctionPlayerResponse{
		ID:                             ap.ID,
		AuctionID:                      ap.AuctionID,
		TournamentPlayerRegistrationID: ap.TournamentPlayerRegistrationID,
		Status:                         ap.Status,
		BasePrice:                      ap.BasePrice,
		FinalPrice:                     ap.FinalPrice,
		SoldToTeamRegistrationID:       ap.SoldToTeamRegistrationID,
		LotNumber:                      ap.LotNumber,
		IsActive:                       ap.IsActive,
		CreatedAt:                      ap.CreatedAt,
	}
}

type BidResponse struct {
	ID                 string                 `json:"id"`
	AuctionPlayerID   string                 `json:"auctionPlayerId"`
	TeamRegistrationID string                `json:"teamRegistrationId"`
	Amount            int64                  `json:"amount"`
	RecordedAt        int64                  `json:"recordedAt"`
}

func toBidResponse(b *auction.Bid) BidResponse {
	if b == nil {
		return BidResponse{}
	}
	return BidResponse{
		ID:                 b.ID,
		AuctionPlayerID:   b.AuctionPlayerID,
		TeamRegistrationID: b.TeamRegistrationID,
		Amount:             b.Amount,
		RecordedAt:         b.RecordedAt,
	}
}

func relayShortTeamLabel(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, sep := range []string{" — ", " – ", " - ", " –", "-"} {
		if i := strings.Index(name, sep); i > 0 && i <= 28 {
			return strings.TrimSpace(name[:i])
		}
	}
	if len(name) > 28 {
		return name[:25] + "…"
	}
	return name
}

func findTournamentTeamByRegID(teams []*auction.TournamentTeamRegistration, teamRegID string) *auction.TournamentTeamRegistration {
	for _, t := range teams {
		if t != nil && t.ID == teamRegID {
			return t
		}
	}
	return nil
}

// RelayLeadingBidderResponse mirrors the “high bid” team card (logo, name, amount shown to audience).
type RelayLeadingBidderResponse struct {
	TeamRegistrationID string `json:"teamRegistrationId,omitempty"`
	TeamName           string `json:"teamName"`
	TeamLogoURL        string `json:"teamLogoUrl"`
	ShortLabel         string `json:"shortLabel"` // compact label for subtitles
	HighBidPaisa       int64  `json:"highBidPaisa"`   // actual top bid; 0 if none
	DisplayPaisa       int64  `json:"displayPaisa"`   // amount to show large (matches console: base until first bid)
	HasBid             bool   `json:"hasBid"`
}

// RelaySnapshotResponse is pushed over SSE for spectator / read-only clients.
type RelaySnapshotResponse struct {
	AuctionID                 string                               `json:"auctionId"`
	TournamentID              string                               `json:"tournamentId"`
	TournamentEventID         string                               `json:"tournamentEventId"`
	Mode                      auction.AuctionMode                  `json:"mode"`
	RunMode                   auction.AuctionRunMode               `json:"runMode"`
	GeneratedAtMs             int64                                `json:"generatedAtMs"`
	FocusLot                  *AuctionPlayerResponse               `json:"focusLot"`
	RegistrationSerial        int                                  `json:"registrationSerial,omitempty"`
	PlayerID                  string                               `json:"playerId,omitempty"`
	LeadingBidder             *RelayLeadingBidderResponse          `json:"leadingBidder,omitempty"`
	Bids                      []BidResponse                        `json:"bids"`
	HighBidPaisa              int64                                `json:"highBidPaisa"`
	LeadingTeamRegistrationID string                               `json:"leadingTeamRegistrationId,omitempty"`
	BidCount                  int                                  `json:"bidCount"`
	Teams                     []TournamentTeamRegistrationResponse `json:"teams"`
	TournamentName            string                               `json:"tournamentName,omitempty"`
	TournamentLogoURL         string                               `json:"tournamentLogoUrl,omitempty"`
	TournamentSponsors        []auction.TournamentSponsorGroup     `json:"tournamentSponsors,omitempty"`
	// TeamBidLimits: per-team max single bid now (derived) + configured cap; updated each relay tick.
	TeamBidLimits []auctionservice.RelayTeamBidLimit `json:"teamBidLimits,omitempty"`
}

func pickFocusLotForRelay(lots []*auction.AuctionPlayer) *auction.AuctionPlayer {
	for _, l := range lots {
		if l != nil && l.IsActive {
			return l
		}
	}
	var best *auction.AuctionPlayer
	for _, l := range lots {
		if l == nil {
			continue
		}
		if l.Status != auction.AuctionPlayerPending && l.Status != auction.AuctionPlayerActive {
			continue
		}
		if best == nil || l.LotNumber < best.LotNumber {
			best = l
		}
	}
	return best
}

func (h *AuctionHandler) buildRelaySnapshot(ctx context.Context, auctionID string, nowMs int64) (RelaySnapshotResponse, error) {
	out := RelaySnapshotResponse{GeneratedAtMs: nowMs, Bids: []BidResponse{}, Teams: []TournamentTeamRegistrationResponse{}}
	auc, err := h.auctionSvc.GetAuction(ctx, auctionID)
	if err != nil {
		return out, err
	}
	if auc == nil {
		return out, fmt.Errorf("auction not found")
	}
	out.AuctionID = auc.ID
	out.TournamentID = auc.TournamentID
	out.TournamentEventID = auc.TournamentEventID
	out.Mode = auc.Mode
	out.RunMode = auc.RunMode

	if tour, terr := h.auctionSvc.GetTournament(ctx, auc.TournamentID); terr == nil && tour != nil {
		out.TournamentName = tour.Name
		out.TournamentLogoURL = tour.LogoURL
		if len(tour.Sponsors) > 0 {
			out.TournamentSponsors = tour.Sponsors
		}
	}

	regTeams, err := h.auctionSvc.ListRegisteredTeams(ctx, auc.TournamentID, auc.TournamentEventID)
	if err != nil {
		return out, err
	}
	for _, t := range regTeams {
		if t == nil {
			continue
		}
		out.Teams = append(out.Teams, TournamentTeamRegistrationResponse{
			ID: t.ID, TournamentID: t.TournamentID, TournamentEventID: t.TournamentEventID,
			TeamName: t.TeamName, TeamLogoURL: t.TeamLogoURL, CreatedAt: t.CreatedAt,
		})
	}

	lots, err := h.auctionPlayerRepo.ListByAuction(ctx, auctionID)
	if err != nil {
		return out, err
	}
	if limits, limErr := h.bidSvc.TeamRelayBidLimits(ctx, auc, lots); limErr == nil {
		out.TeamBidLimits = limits
	}

	var focus *auction.AuctionPlayer
	if id := strings.TrimSpace(auc.DisplayAuctionPlayerID); id != "" {
		for _, l := range lots {
			if l != nil && l.ID == id {
				focus = l
				break
			}
		}
	}
	if focus == nil {
		focus = pickFocusLotForRelay(lots)
	}
	if focus == nil {
		return out, nil
	}
	fl := toAuctionPlayerResponse(focus)
	out.FocusLot = &fl

	if serial, playerID, metaErr := h.lotSvc.RegistrationDisplayMeta(ctx, focus.TournamentPlayerRegistrationID); metaErr == nil {
		out.RegistrationSerial = serial
		out.PlayerID = playerID
	}

	bids, err := h.bidRepo.ListByAuctionPlayer(ctx, focus.ID)
	if err != nil {
		return out, err
	}
	sort.Slice(bids, func(i, j int) bool {
		if bids[i] == nil || bids[j] == nil {
			return false
		}
		return bids[i].RecordedAt < bids[j].RecordedAt
	})
	const maxBids = 50
	start := 0
	if len(bids) > maxBids {
		start = len(bids) - maxBids
	}
	for i := start; i < len(bids); i++ {
		if bids[i] == nil {
			continue
		}
		out.Bids = append(out.Bids, toBidResponse(bids[i]))
	}
	out.BidCount = len(bids)
	var highPaisa int64
	var leadTeamID string
	if hb, hbErr := h.bidRepo.GetHighestBid(ctx, focus.ID); hbErr == nil && hb != nil {
		highPaisa = hb.Amount
		leadTeamID = hb.TeamRegistrationID
		out.HighBidPaisa = highPaisa
		out.LeadingTeamRegistrationID = leadTeamID
	}
	hasBid := out.BidCount > 0 && highPaisa > 0
	displayPaisa := focus.BasePrice
	if hasBid {
		displayPaisa = highPaisa
	}
	lb := &RelayLeadingBidderResponse{
		TeamRegistrationID: leadTeamID,
		HighBidPaisa:       highPaisa,
		DisplayPaisa:       displayPaisa,
		HasBid:             hasBid,
	}
	if leadTeamID != "" {
		if tr := findTournamentTeamByRegID(regTeams, leadTeamID); tr != nil {
			lb.TeamName = tr.TeamName
			lb.TeamLogoURL = tr.TeamLogoURL
			lb.ShortLabel = relayShortTeamLabel(tr.TeamName)
		}
	}
	out.LeadingBidder = lb
	return out, nil
}

// GET /v1/auctions/:auctionId/relay/stream — Server-Sent Events; JSON snapshots ~1s for spectators.
func (h *AuctionHandler) StreamAuctionRelay(c *gin.Context) {
	auctionID := strings.TrimSpace(c.Param("auctionId"))
	if auctionID == "" {
		Error(c, http.StatusBadRequest, "Validation Error", "auctionId is required")
		return
	}
	if _, err := h.auctionSvc.GetAuction(c.Request.Context(), auctionID); err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		Error(c, http.StatusInternalServerError, "Internal Server Error", "streaming unsupported")
		return
	}

	c.Status(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(900 * time.Millisecond)
	defer ticker.Stop()

	writeSnapshot := func() bool {
		snap, err := h.buildRelaySnapshot(c.Request.Context(), auctionID, time.Now().UnixMilli())
		if err != nil {
			return false
		}
		b, err := json.Marshal(snap)
		if err != nil {
			return false
		}
		_, _ = fmt.Fprintf(c.Writer, "event: snapshot\ndata: %s\n\n", b)
		flusher.Flush()
		return true
	}

	if !writeSnapshot() {
		return
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			if !writeSnapshot() {
				return
			}
		}
	}
}

type TournamentPlayerRegistrationResponse struct {
	ID                string `json:"id"`
	TournamentID      string `json:"tournamentId"`
	TournamentEventID string `json:"tournamentEventId"`
	PlayerID         string `json:"playerId"`
	TeamID           string `json:"teamId"`
	SerialNumber      int    `json:"serialNumber"`
	CreatedAt         int64  `json:"createdAt"`
}

type TournamentTeamRegistrationResponse struct {
	ID                string `json:"id"`
	TournamentID      string `json:"tournamentId"`
	TournamentEventID string `json:"tournamentEventId"`
	TeamName          string `json:"teamName"`
	TeamLogoURL       string `json:"teamLogoUrl"`
	CreatedAt         int64  `json:"createdAt"`
}

type WalletResponse struct {
	ID                string `json:"id"`
	TournamentID      string `json:"tournamentId"`
	TournamentEventID string `json:"tournamentEventId"`
	TeamID            string `json:"teamId"`
	Balance           int64  `json:"balance"`
	MaxBidAmount      int64  `json:"maxBidAmount"`
	// Set when GET …/wallets includes auctionId (same semantics as POST bid bidHints).
	DerivedMaxBidAmount int64 `json:"derivedMaxBidAmount,omitempty"`
	MinBidAmount        int64 `json:"minBidAmount,omitempty"`
	CreatedAt           int64  `json:"createdAt"`
	UpdatedAt           int64  `json:"updatedAt"`
}

func toWalletResponse(w *auction.Wallet) WalletResponse {
	if w == nil {
		return WalletResponse{}
	}
	return WalletResponse{
		ID: w.ID, TournamentID: w.TournamentID, TournamentEventID: w.TournamentEventID, TeamID: w.TeamID,
		Balance: w.Balance, MaxBidAmount: w.MaxBidAmount, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
}

// --- Handlers ---

// resolveRegistrationFilter applies stateless category selection:
// - If presetId is set, load the auction and use the matching saved FilterPreset.Filter.
// - Otherwise use the inline filter from the request body.
func (h *AuctionHandler) resolveRegistrationFilter(c *gin.Context, auctionID string, presetID string, body RegistrationFilterRequest) (auction.RegistrationFilter, error) {
	if strings.TrimSpace(presetID) != "" {
		auc, err := h.auctionSvc.GetAuction(c.Request.Context(), auctionID)
		if err != nil {
			return auction.RegistrationFilter{}, err
		}
		f, ok := auc.FilterForPreset(presetID)
		if !ok {
			return auction.RegistrationFilter{}, &auctionservice.ValidationError{Err: fmt.Errorf("filter preset not found: %s", presetID)}
		}
		return f, nil
	}
	return body.toDomain(), nil
}

// GET /v1/auctions/:auctionId — load auction including filterPresets (stateless category definitions).
func (h *AuctionHandler) GetAuction(c *gin.Context) {
	id := strings.TrimSpace(c.Param("auctionId"))
	auc, err := h.auctionSvc.GetAuction(c.Request.Context(), id)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"auction": auc})
}

// PUT /v1/auctions/:auctionId/display-lot — body: { "auctionPlayerId": "<uuid>" | "" } sets console + relay focus (empty clears).
func (h *AuctionHandler) PutAuctionDisplayLot(c *gin.Context) {
	auctionID := strings.TrimSpace(c.Param("auctionId"))
	var req struct {
		AuctionPlayerID string `json:"auctionPlayerId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	auc, err := h.lotSvc.SetDisplayAuctionPlayer(c.Request.Context(), auctionID, req.AuctionPlayerID)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"auction": auc})
}

// PATCH /v1/auctions/:auctionId — body: { "runMode": "test" | "live" }
func (h *AuctionHandler) PatchAuction(c *gin.Context) {
	auctionID := strings.TrimSpace(c.Param("auctionId"))
	var req struct {
		RunMode string `json:"runMode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	if strings.TrimSpace(req.RunMode) == "" {
		Error(c, http.StatusBadRequest, "Validation Error", "runMode is required")
		return
	}
	auc, err := h.auctionSvc.UpdateAuctionRunMode(c.Request.Context(), auctionID, auction.AuctionRunMode(strings.TrimSpace(req.RunMode)))
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"auction": auc})
}

// GET /v1/tournaments/:tournamentId — tournament metadata (logo, sponsors JSON array).
func (h *AuctionHandler) GetTournament(c *gin.Context) {
	id := strings.TrimSpace(c.Param("tournamentId"))
	t, err := h.auctionSvc.GetTournament(c.Request.Context(), id)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "tournament not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tournament": t})
}

// GET /v1/tournaments/:tournamentId/events/:tournamentEventId/teams
func (h *AuctionHandler) ListRegisteredTeams(c *gin.Context) {
	tournamentID := strings.TrimSpace(c.Param("tournamentId"))
	tournamentEventID := strings.TrimSpace(c.Param("tournamentEventId"))
	teams, err := h.auctionSvc.ListRegisteredTeams(c.Request.Context(), tournamentID, tournamentEventID)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "tournament event not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	items := make([]TournamentTeamRegistrationResponse, 0, len(teams))
	for _, t := range teams {
		if t == nil {
			continue
		}
		items = append(items, TournamentTeamRegistrationResponse{
			ID: t.ID, TournamentID: t.TournamentID, TournamentEventID: t.TournamentEventID,
			TeamName: t.TeamName, TeamLogoURL: t.TeamLogoURL, CreatedAt: t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"teams": items})
}

// POST /api/v1/auctions
func (h *AuctionHandler) CreateAuction(c *gin.Context) {
	var req CreateAuctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	mode := auction.AuctionMode(req.Mode)
	if req.Mode == "" {
		mode = auction.AuctionModeOpen
	}
	runMode := auction.AuctionRunMode(req.RunMode)
	if req.RunMode == "" {
		runMode = auction.AuctionRunModeTest
	}

	// CreateAuction service expects filter presets already including CreatedAt (DB stored as JSONB).
	// For MVP, we just set CreatedAt = 0 here; repositories persist what services return.
	filterPresets := make([]auction.FilterPreset, 0, len(req.FilterPresets))
	for _, p := range req.FilterPresets {
		filterPresets = append(filterPresets, auction.FilterPreset{
			ID:        p.ID,
			Name:      p.Name,
			Filter:    p.Filter.toDomain(),
			CreatedAt: 0,
		})
	}

	rules := auction.AuctionRules{
		MinBidAmount: req.Rules.MinBidAmount,
		MaxBidAmount: req.Rules.MaxBidAmount,
	}
	auc, err := h.auctionSvc.CreateAuction(c.Request.Context(), req.TournamentID, req.TournamentEventID, mode, runMode, filterPresets, rules)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"auction": gin.H{
		"id":                auc.ID,
		"tournamentId":      auc.TournamentID,
		"tournamentEventId": auc.TournamentEventID,
		"mode":              auc.Mode,
		"runMode":           auc.RunMode,
		"filterPresets":     auc.FilterPresets,
		"rules":             auc.Rules,
		"createdAt":         auc.CreatedAt,
	}})
}

// POST /v1/auctions/:auctionId/reset — test auctions only: revert sales, clear bids, all lots pending.
func (h *AuctionHandler) ResetTestAuction(c *gin.Context) {
	auctionID := strings.TrimSpace(c.Param("auctionId"))
	err := h.settlementSvc.ResetTestAuction(c.Request.Context(), auctionID)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "only allowed when auction runMode is test") {
				Error(c, http.StatusForbidden, "Forbidden", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/v1/auctions/:auctionId/eligible
func (h *AuctionHandler) ListEligibleRegistrations(c *gin.Context) {
	auctionID := c.Param("auctionId")
	var req ListEligibleRequest
	// body optional: empty filter means no filter
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ListEligibleRequest{Filter: RegistrationFilterRequest{}}
	}

	filter, err := h.resolveRegistrationFilter(c, auctionID, req.PresetID, req.Filter)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	regs, err := h.lotSvc.ListEligibleRegistrations(c.Request.Context(), auctionID, filter)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	items := make([]TournamentPlayerRegistrationResponse, 0, len(regs))
	for _, r := range regs {
		if r == nil {
			continue
		}
		items = append(items, toTournamentPlayerRegistrationResponse(r))
	}
	c.JSON(http.StatusOK, gin.H{"registrations": items})
}

// POST /api/v1/auctions/:auctionId/lots/bulk
func (h *AuctionHandler) CreateLotsBulk(c *gin.Context) {
	auctionID := c.Param("auctionId")
	var req CreateLotsBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	var lots []*auction.AuctionPlayer
	var err error
	if len(req.RegistrationIDs) > 0 {
		// Backward-compatible path: client provides explicit registration IDs.
		lots, err = h.lotSvc.CreateLotsBulk(c.Request.Context(), auctionservice.CreateLotsBulkInput{
			AuctionID:        auctionID,
			RegistrationIDs: req.RegistrationIDs,
			StartLotNumber:  req.StartLotNumber,
			BasePrice:        req.BasePrice,
		})
	} else {
		filter, rerr := h.resolveRegistrationFilter(c, auctionID, req.PresetID, req.Filter)
		if rerr != nil {
			if auctionservice.IsValidationError(rerr) {
				Error(c, http.StatusBadRequest, "Validation Error", rerr.Error())
				return
			}
			Error(c, http.StatusInternalServerError, "Internal Server Error", rerr.Error())
			return
		}
		// MVP path: create lots by server-side lookup of eligible registrations.
		lots, err = h.lotSvc.CreateLotsByQuery(c.Request.Context(), auctionservice.CreateLotsByQueryInput{
			AuctionID:          auctionID,
			TournamentID:      req.TournamentID,
			TournamentEventID: req.TournamentEventID,
			Filter:            filter,
			StartLotNumber:    req.StartLotNumber,
			BasePrice:         req.BasePrice,
			Limit:              req.Limit,
		})
	}
	if err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	resp := make([]AuctionPlayerResponse, 0, len(lots))
	for _, ap := range lots {
		resp = append(resp, toAuctionPlayerResponse(ap))
	}
	c.JSON(http.StatusCreated, gin.H{"lots": resp})
}

// GET /api/v1/auctions/:auctionId/lots
func (h *AuctionHandler) ListLotsByAuction(c *gin.Context) {
	auctionID := c.Param("auctionId")
	lots, err := h.auctionPlayerRepo.ListByAuction(c.Request.Context(), auctionID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	resp := make([]AuctionPlayerResponse, 0, len(lots))
	for _, ap := range lots {
		resp = append(resp, toAuctionPlayerResponse(ap))
	}
	c.JSON(http.StatusOK, gin.H{"lots": resp})
}

// GET /v1/auctions/:auctionId/lots/by-registration-serial/:serialNumber
// Returns the lot for that auction + registration serial (any status: sold, unsold, active, etc.).
func (h *AuctionHandler) GetLotByRegistrationSerial(c *gin.Context) {
	auctionID := strings.TrimSpace(c.Param("auctionId"))
	serialStr := strings.TrimSpace(c.Param("serialNumber"))
	serial, err := strconv.Atoi(serialStr)
	if err != nil || serial < 1 {
		Error(c, http.StatusBadRequest, "Validation Error", "serialNumber must be a positive integer")
		return
	}
	ap, reg, err := h.lotSvc.GetLotByRegistrationSerial(c.Request.Context(), auctionID, serial)
	if err != nil {
		if auctionservice.IsValidationError(err) {
			if strings.Contains(err.Error(), "auction not found") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "no lot for") {
				Error(c, http.StatusNotFound, "Not Found", err.Error())
				return
			}
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"lot":            toAuctionPlayerResponse(ap),
		"serialNumber":   reg.SerialNumber,
		"playerId":       reg.PlayerID,
		"registration":   toTournamentPlayerRegistrationResponse(reg),
	})
}

func toTournamentPlayerRegistrationResponse(r *auction.TournamentPlayerRegistration) TournamentPlayerRegistrationResponse {
	if r == nil {
		return TournamentPlayerRegistrationResponse{}
	}
	return TournamentPlayerRegistrationResponse{
		ID: r.ID, TournamentID: r.TournamentID, TournamentEventID: r.TournamentEventID,
		PlayerID: r.PlayerID, TeamID: r.TeamID, SerialNumber: r.SerialNumber, CreatedAt: r.CreatedAt,
	}
}

// POST /api/v1/auction-players/:auctionPlayerId/bids
func (h *AuctionHandler) PlaceBid(c *gin.Context) {
	auctionPlayerID := c.Param("auctionPlayerId")
	var req PlaceBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	bid, bidHints, err := h.bidSvc.PlaceBid(c.Request.Context(), auctionservice.PlaceBidInput{
		AuctionPlayerID:    auctionPlayerID,
		TeamRegistrationID: req.TeamRegistrationID,
		Amount:             req.Amount,
	})
	if err != nil {
		var ve *auctionservice.ValidationError
		if errors.As(err, &ve) && ve.Hints != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"type":   "https://api.dreamers.be/errors/400",
					"title":  "Validation Error",
					"status": http.StatusBadRequest,
					"detail": ve.Err.Error(),
				},
				"bidHints": ve.Hints,
			})
			return
		}
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"bid": toBidResponse(bid), "bidHints": bidHints})
}

// GET /api/v1/auction-players/:auctionPlayerId/bids
func (h *AuctionHandler) ListBidsByAuctionPlayer(c *gin.Context) {
	auctionPlayerID := c.Param("auctionPlayerId")
	bids, err := h.bidRepo.ListByAuctionPlayer(c.Request.Context(), auctionPlayerID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	resp := make([]BidResponse, 0, len(bids))
	for _, b := range bids {
		resp = append(resp, toBidResponse(b))
	}
	c.JSON(http.StatusOK, gin.H{"bids": resp})
}

// POST /api/v1/auction-players/:auctionPlayerId/sell
func (h *AuctionHandler) SellCurrentLot(c *gin.Context) {
	auctionPlayerID := c.Param("auctionPlayerId")
	var req SellRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.settlementSvc.SellCurrentLot(c.Request.Context(), auctionservice.SellInput{
		AuctionPlayerID: auctionPlayerID,
		ExpectedBidID:   req.ExpectedBidID,
	}); err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/v1/auction-players/:auctionPlayerId/unsold
func (h *AuctionHandler) MarkUnsold(c *gin.Context) {
	auctionPlayerID := c.Param("auctionPlayerId")
	if err := h.settlementSvc.MarkUnsold(c.Request.Context(), auctionPlayerID); err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/v1/auction-players/:auctionPlayerId/revert
func (h *AuctionHandler) RevertSale(c *gin.Context) {
	auctionPlayerID := c.Param("auctionPlayerId")
	var req RevertSaleRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.settlementSvc.RevertSale(c.Request.Context(), auctionPlayerID, req.Reason); err != nil {
		if auctionservice.IsValidationError(err) {
			Error(c, http.StatusBadRequest, "Validation Error", err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/v1/wallets?teamRegistrationId=&tournamentId=&tournamentEventId=&auctionId=
// Optional auctionId: includes derivedMaxBidAmount (max single bid now given balance + min roster at min bid, capped by auction/team max).
func (h *AuctionHandler) GetWallet(c *gin.Context) {
	teamRegistrationID := c.Query("teamRegistrationId")
	tournamentID := c.Query("tournamentId")
	tournamentEventID := c.Query("tournamentEventId")
	auctionID := strings.TrimSpace(c.Query("auctionId"))

	if teamRegistrationID == "" || tournamentID == "" || tournamentEventID == "" {
		Error(c, http.StatusBadRequest, "Validation Error", "teamRegistrationId, tournamentId, tournamentEventId are required")
		return
	}

	w, err := h.walletRepo.GetByTournamentEventTeam(c.Request.Context(), tournamentID, tournamentEventID, teamRegistrationID)
	if err != nil {
		Error(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	if w == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	wr := toWalletResponse(w)
	if auctionID != "" {
		if auc, aerr := h.auctionSvc.GetAuction(c.Request.Context(), auctionID); aerr == nil && auc != nil {
			if auc.TournamentID == w.TournamentID && auc.TournamentEventID == w.TournamentEventID {
				wr.MinBidAmount = auc.Rules.MinBidAmount
			}
		}
		if d, derr := h.bidSvc.DerivedMaxBidForTeam(c.Request.Context(), auctionID, teamRegistrationID, w); derr == nil {
			wr.DerivedMaxBidAmount = d
		}
	}
	c.JSON(http.StatusOK, gin.H{"wallet": wr})
}

// helper: not currently used but kept for future URL query parsing
func parseIntQuery(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

