package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/domain"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/service"
)

// SessionHandler exposes session-related routes.
type SessionHandler struct {
	svc *service.SessionService
}

// NewSessionHandler constructs a SessionHandler backed by the given service.
func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// ── Route registration ─────────────────────────────────────────────────────────

// RegisterUnderSeminar wires the session-creation route onto the seminars group.
// Expected prefix: /v1/seminars/:id
func (h *SessionHandler) RegisterUnderSeminar(rg *gin.RouterGroup) {
	rg.GET("/:id/sessions", h.List)
	rg.POST("/:id/sessions", h.Create)
}

// Register wires top-level session routes onto the provided router group.
// Expected prefix: /v1/sessions
func (h *SessionHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id", h.Get)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/abandon", h.Abandon)
	rg.POST("/:id/residue", h.SubmitResidue)
}

// ── Handlers ───────────────────────────────────────────────────────────────────

// Create godoc
//
//	@Summary  Create a session for a seminar
//	@Tags     sessions
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                          true  "Seminar ID"
//	@Param    body  body      apphttp.CreateSessionRequest    true  "session fields"
//	@Success  201   {object}  apphttp.SessionResponse
//	@Router   /v1/seminars/{id}/sessions [post]
func (h *SessionHandler) Create(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	sess, err := h.svc.Create(c.Request.Context(), ownerSub, c.Param("id"), service.CreateSessionParams{
		SectionLabel: req.SectionLabel,
		Mode:         req.Mode,
		ExcerptText:  req.ExcerptText,
		ReconMinutes: req.ReconMinutes,
	})
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toSessionResponse(*sess))
}

// List godoc
//
//	@Summary  List sessions for a seminar
//	@Tags     sessions
//	@Produce  json
//	@Param    id   path      string  true  "Seminar ID"
//	@Success  200  {array}   apphttp.SessionResponse
//	@Router   /v1/seminars/{id}/sessions [get]
func (h *SessionHandler) List(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sessions, err := h.svc.List(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	responses := make([]apphttp.SessionResponse, len(sessions))
	for i, s := range sessions {
		responses[i] = toSessionResponse(s)
	}
	c.JSON(http.StatusOK, responses)
}

// Delete godoc
//
//	@Summary  Delete a session
//	@Tags     sessions
//	@Param    id   path      string  true  "Session ID"
//	@Success  204
//	@Router   /v1/sessions/{id} [delete]
func (h *SessionHandler) Delete(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	err = h.svc.Delete(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Get godoc
//
//	@Summary  Get a session with its turns
//	@Tags     sessions
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {object}  apphttp.SessionDetailResponse
//	@Router   /v1/sessions/{id} [get]
func (h *SessionHandler) Get(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	detail, err := h.svc.Get(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSessionDetailResponse(detail))
}

// Abandon godoc
//
//	@Summary  Abandon an in-progress session
//	@Tags     sessions
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {object}  apphttp.SessionResponse
//	@Router   /v1/sessions/{id}/abandon [post]
func (h *SessionHandler) Abandon(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sess, err := h.svc.Abandon(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSessionResponse(*sess))
}

// SubmitResidue godoc
//
//	@Summary  Submit residue text to complete a session
//	@Tags     sessions
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                        true  "Session ID"
//	@Param    body  body      apphttp.SubmitResidueRequest  true  "residue text"
//	@Success  200   {object}  apphttp.SessionResponse
//	@Router   /v1/sessions/{id}/residue [post]
func (h *SessionHandler) SubmitResidue(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.SubmitResidueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	sess, err := h.svc.SubmitResidue(c.Request.Context(), c.Param("id"), ownerSub, req.ResidueText)
	if err != nil {
		handleSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSessionResponse(*sess))
}

// ── helpers ────────────────────────────────────────────────────────────────────

// toSessionResponse converts a domain.Session to its HTTP response shape.
func toSessionResponse(s domain.Session) apphttp.SessionResponse {
	return apphttp.SessionResponse{
		ID:             s.ID,
		SeminarID:      s.SeminarID,
		SectionLabel:   s.SectionLabel,
		Mode:           s.Mode,
		ExcerptText:    s.ExcerptText,
		ExcerptHash:    s.ExcerptHash,
		Status:         s.Status,
		Phase:          s.Phase,
		ReconMinutes:   s.ReconMinutes,
		PhaseStartedAt: s.PhaseStartedAt,
		PhaseEndsAt:    s.PhaseEndsAt,
		StartedAt:      s.StartedAt,
		EndedAt:        s.EndedAt,
		ResidueText:    s.ResidueText,
	}
}

// toTurnResponse converts a domain.Turn to its HTTP response shape.
func toTurnResponse(t domain.Turn) apphttp.TurnResponse {
	return apphttp.TurnResponse{
		ID:        t.ID,
		SessionID: t.SessionID,
		Phase:     t.Phase,
		Speaker:   t.Speaker,
		Text:      t.Text,
		Flags:     t.Flags,
		CreatedAt: t.CreatedAt,
	}
}

// toSessionDetailResponse converts a SessionDetail to its HTTP response shape.
func toSessionDetailResponse(d *service.SessionDetail) apphttp.SessionDetailResponse {
	turns := make([]apphttp.TurnResponse, len(d.Turns))
	for i, t := range d.Turns {
		turns[i] = toTurnResponse(t)
	}
	return apphttp.SessionDetailResponse{
		SessionResponse: toSessionResponse(*d.Session),
		Turns:           turns,
	}
}

// handleSessionServiceError extends handleServiceError with session-specific
// typed errors.
func handleSessionServiceError(c *gin.Context, err error) {
	var terminal *service.ErrSessionTerminalError
	if errors.As(err, &terminal) {
		apphttp.Fail(c, http.StatusConflict, "session_terminal",
			err.Error())
		return
	}

	var phaseExpired *service.ErrPhaseExpiredError
	if errors.As(err, &phaseExpired) {
		apphttp.Fail(c, http.StatusUnprocessableEntity, "phase_expired",
			err.Error())
		return
	}

	var phaseNoTurns *service.ErrPhaseNoTurnsError
	if errors.As(err, &phaseNoTurns) {
		apphttp.Fail(c, http.StatusUnprocessableEntity, "phase_no_turns",
			err.Error())
		return
	}

	// Fall back to the seminar error handler which covers NotFound and Validation.
	handleServiceError(c, err)
}
