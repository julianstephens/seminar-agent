package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/domain"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/modules/tutorial/service"
)

// TutorialSessionHandler exposes top-level tutorial session routes.
type TutorialSessionHandler struct {
	sessionSvc  *service.TutorialSessionService
	artifactSvc *service.ArtifactService
	turnSvc     *service.TutorialTurnService
}

// NewTutorialSessionHandler constructs a TutorialSessionHandler.
func NewTutorialSessionHandler(
	sessionSvc *service.TutorialSessionService,
	artifactSvc *service.ArtifactService,
	turnSvc *service.TutorialTurnService,
) *TutorialSessionHandler {
	return &TutorialSessionHandler{
		sessionSvc:  sessionSvc,
		artifactSvc: artifactSvc,
		turnSvc:     turnSvc,
	}
}

// ── Route registration ─────────────────────────────────────────────────────────

// Register wires top-level tutorial session routes onto the provided router group.
// Expected prefix: /v1/tutorial-sessions
func (h *TutorialSessionHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id", h.Get)
	rg.DELETE("/:id", h.Delete)
	rg.POST("/:id/complete", h.Complete)
	rg.POST("/:id/abandon", h.Abandon)
	rg.GET("/:id/problem-set", h.GetSessionProblemSet)
	rg.DELETE("/:id/problem-set", h.DeleteSessionProblemSet)
	rg.GET("/:id/artifacts", h.ListArtifacts)
	rg.POST("/:id/artifacts", h.CreateArtifact)
	rg.DELETE("/:id/artifacts/:artifactId", h.DeleteArtifact)
	rg.POST("/:id/turns", h.SubmitTurn)
	rg.GET("/:id/turns", h.ListTurns)
}

// ── Session Handlers ───────────────────────────────────────────────────────────

// Get godoc
//
//	@Summary  Get a tutorial session with its artifacts and turns
//	@Tags     tutorial-sessions
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {object}  apphttp.TutorialSessionDetailResponse
//	@Router   /v1/tutorial-sessions/{id} [get]
func (h *TutorialSessionHandler) Get(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sessionID := c.Param("id")
	_ = c.Error(fmt.Errorf("[TutorialSessionHandler.Get] Fetching session_id=%s owner=%s", sessionID, ownerSub))

	detail, err := h.sessionSvc.GetTutorialSession(c.Request.Context(), sessionID, ownerSub)
	if err != nil {
		_ = c.Error(fmt.Errorf("[TutorialSessionHandler.Get] Failed to get session: %w", err))
		handleServiceError(c, err)
		return
	}

	artifacts := make([]apphttp.ArtifactResponse, len(detail.Artifacts))
	for i, a := range detail.Artifacts {
		artifacts[i] = toArtifactResponse(a)
	}

	turns := make([]apphttp.TutorialTurnResponse, len(detail.Turns))
	for i, t := range detail.Turns {
		turns[i] = toTutorialTurnResponse(t)
	}

	resp := apphttp.TutorialSessionDetailResponse{
		TutorialSessionResponse: toTutorialSessionResponse(*detail.Session),
		Artifacts:               artifacts,
		Turns:                   turns,
	}
	if detail.ProblemSet != nil {
		ps := toProblemSetResponse(*detail.ProblemSet)
		resp.ProblemSet = &ps
	}
	c.JSON(http.StatusOK, resp)
}

// Delete godoc
//
//	@Summary  Delete a tutorial session
//	@Tags     tutorial-sessions
//	@Param    id   path  string  true  "Session ID"
//	@Success  204
//	@Router   /v1/tutorial-sessions/{id} [delete]
func (h *TutorialSessionHandler) Delete(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	if err := h.sessionSvc.DeleteTutorialSession(c.Request.Context(), c.Param("id"), ownerSub); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Complete godoc
//
//	@Summary  Mark a tutorial session as complete
//	@Tags     tutorial-sessions
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                                    true   "Session ID"
//	@Param    body  body      apphttp.CompleteTutorialSessionRequest    false  "optional notes"
//	@Success  200   {object}  apphttp.TutorialSessionResponse
//	@Router   /v1/tutorial-sessions/{id}/complete [post]
func (h *TutorialSessionHandler) Complete(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CompleteTutorialSessionRequest
	// Notes are optional; ignore bind errors.
	_ = c.ShouldBindJSON(&req)

	sess, err := h.sessionSvc.CompleteTutorialSession(c.Request.Context(), c.Param("id"), ownerSub, req.Notes)
	if err != nil {
		handleTutorialSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTutorialSessionResponse(*sess))
}

// Abandon godoc
//
//	@Summary  Abandon an in-progress tutorial session
//	@Tags     tutorial-sessions
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {object}  apphttp.TutorialSessionResponse
//	@Router   /v1/tutorial-sessions/{id}/abandon [post]
func (h *TutorialSessionHandler) Abandon(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sess, err := h.sessionSvc.AbandonTutorialSession(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleTutorialSessionServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTutorialSessionResponse(*sess))
}

// ── Problem Set Handlers ───────────────────────────────────────────────────────

// GetSessionProblemSet godoc
//
//	@Summary  Get the problem set assigned from a tutorial session
//	@Tags     tutorial-sessions
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {object}  apphttp.ProblemSetResponse
//	@Success  404
//	@Router   /v1/tutorial-sessions/{id}/problem-set [get]
func (h *TutorialSessionHandler) GetSessionProblemSet(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	// First verify session ownership
	_, err = h.sessionSvc.GetTutorialSession(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Directly access repo through session service's repo
	// We need to get the repo from somewhere - let's check TutorialSessionHandler structure
	ps, err := h.sessionSvc.GetSessionProblemSet(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toProblemSetResponse(*ps))
}

// DeleteSessionProblemSet godoc
//
//	@Summary  Soft delete a problem set assigned from a tutorial session
//	@Tags     tutorial-sessions
//	@Param    id   path  string  true  "Session ID"
//	@Success  204
//	@Router   /v1/tutorial-sessions/{id}/problem-set [delete]
func (h *TutorialSessionHandler) DeleteSessionProblemSet(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	// First verify session ownership and get problem set
	ps, err := h.sessionSvc.GetSessionProblemSet(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Soft delete by updating status
	err = h.sessionSvc.DeleteSessionProblemSet(c.Request.Context(), ps.ID, ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Artifact Handlers ──────────────────────────────────────────────────────────

// ListArtifacts godoc
//
//	@Summary  List artifacts for a tutorial session
//	@Tags     artifacts
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {array}   apphttp.ArtifactResponse
//	@Router   /v1/tutorial-sessions/{id}/artifacts [get]
func (h *TutorialSessionHandler) ListArtifacts(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	artifacts, err := h.artifactSvc.ListArtifacts(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	resp := make([]apphttp.ArtifactResponse, len(artifacts))
	for i, a := range artifacts {
		resp[i] = toArtifactResponse(a)
	}
	c.JSON(http.StatusOK, resp)
}

// CreateArtifact godoc
//
//	@Summary  Create an artifact for a tutorial session
//	@Tags     artifacts
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                        true  "Session ID"
//	@Param    body  body      apphttp.CreateArtifactRequest true  "artifact fields"
//	@Success  201   {object}  apphttp.ArtifactResponse
//	@Router   /v1/tutorial-sessions/{id}/artifacts [post]
func (h *TutorialSessionHandler) CreateArtifact(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CreateArtifactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	art, err := h.artifactSvc.CreateArtifact(c.Request.Context(), ownerSub, c.Param("id"), service.CreateArtifactParams{
		Kind:         domain.ArtifactKind(req.Kind),
		Title:        req.Title,
		Content:      req.Content,
		ProblemSetID: req.ProblemSetID,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toArtifactResponse(*art))
}

// DeleteArtifact godoc
//
//	@Summary  Delete an artifact
//	@Tags     artifacts
//	@Param    id          path  string  true  "Session ID"
//	@Param    artifactId  path  string  true  "Artifact ID"
//	@Success  204
//	@Router   /v1/tutorial-sessions/{id}/artifacts/{artifactId} [delete]
func (h *TutorialSessionHandler) DeleteArtifact(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	if err := h.artifactSvc.DeleteArtifact(c.Request.Context(), c.Param("artifactId"), ownerSub); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Tutorial Turn Handlers ─────────────────────────────────────────────────────

// SubmitTurn godoc
//
//	@Summary  Submit a user turn in a tutorial session
//	@Tags     tutorial-turns
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                                true  "Session ID"
//	@Param    body  body      apphttp.SubmitTutorialTurnRequest     true  "turn text"
//	@Success  201   {object}  apphttp.SubmitTutorialTurnResponse
//	@Router   /v1/tutorial-sessions/{id}/turns [post]
func (h *TutorialSessionHandler) SubmitTurn(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.SubmitTutorialTurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.turnSvc.SubmitTutorialTurn(c.Request.Context(), c.Param("id"), ownerSub, req.Text)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	if result == nil || result.UserTurn == nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error",
			"invalid service response: missing user turn")
		return
	}

	resp := apphttp.SubmitTutorialTurnResponse{
		UserTurn: toTutorialTurnResponse(*result.UserTurn),
	}
	if result.AgentTurn != nil {
		ar := toTutorialTurnResponse(*result.AgentTurn)
		resp.AgentTurn = &ar
	}

	c.JSON(http.StatusCreated, resp)
}

// ListTurns godoc
//
//	@Summary  List all turns in a tutorial session
//	@Tags     tutorial-turns
//	@Produce  json
//	@Param    id   path      string  true  "Session ID"
//	@Success  200  {array}   apphttp.TutorialTurnResponse
//	@Router   /v1/tutorial-sessions/{id}/turns [get]
func (h *TutorialSessionHandler) ListTurns(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	turns, err := h.turnSvc.ListTutorialTurns(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	resp := make([]apphttp.TutorialTurnResponse, len(turns))
	for i, t := range turns {
		resp[i] = toTutorialTurnResponse(t)
	}
	c.JSON(http.StatusOK, resp)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// handleSessionServiceError extends handleServiceError with tutorial-session-specific typed errors.
func handleSessionServiceError(c *gin.Context, err error) {
	var terminal *service.ErrSessionTerminalError
	if errors.As(err, &terminal) {
		apphttp.Fail(c, http.StatusConflict, "session_terminal", err.Error())
		return
	}

	// Fall back to the generic error handler which covers NotFound and Validation.
	handleServiceError(c, err)
}

// handleTutorialSessionServiceError maps tutorial-session-specific errors to HTTP codes.
func handleTutorialSessionServiceError(c *gin.Context, err error) {
	handleSessionServiceError(c, err)
}
