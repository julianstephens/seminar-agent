package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/domain"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/modules/tutorial/service"
)

// TutorialHandler exposes tutorial-related routes.
type TutorialHandler struct {
	svc        *service.TutorialService
	sessionSvc *service.TutorialSessionService
}

// NewTutorialHandler constructs a TutorialHandler backed by the given services.
func NewTutorialHandler(svc *service.TutorialService, sessionSvc *service.TutorialSessionService) *TutorialHandler {
	return &TutorialHandler{svc: svc, sessionSvc: sessionSvc}
}

// ── Route registration ─────────────────────────────────────────────────────────

// Register wires all tutorial routes onto the provided router group.
func (h *TutorialHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:id", h.Get)
	rg.PATCH("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

// RegisterSessionsUnderTutorial wires the session creation/listing routes
// under /v1/tutorials/:id.
func (h *TutorialHandler) RegisterSessionsUnderTutorial(rg *gin.RouterGroup) {
	rg.GET("/:id/sessions", h.ListSessions)
	rg.POST("/:id/sessions", h.CreateSession)
}

// ── Tutorial Handlers ──────────────────────────────────────────────────────────

// List godoc
//
//	@Summary  List tutorials
//	@Tags     tutorials
//	@Produce  json
//	@Success  200  {array}   apphttp.TutorialResponse
//	@Router   /v1/tutorials [get]
func (h *TutorialHandler) List(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	tutorials, err := h.svc.ListTutorials(c.Request.Context(), ownerSub)
	if err != nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "failed to list tutorials")
		return
	}

	resp := make([]apphttp.TutorialResponse, len(tutorials))
	for i, t := range tutorials {
		resp[i] = toTutorialResponse(t)
	}
	c.JSON(http.StatusOK, resp)
}

// Create godoc
//
//	@Summary  Create a tutorial
//	@Tags     tutorials
//	@Accept   json
//	@Produce  json
//	@Param    body  body      apphttp.CreateTutorialRequest  true  "tutorial fields"
//	@Success  201   {object}  apphttp.TutorialResponse
//	@Router   /v1/tutorials [post]
func (h *TutorialHandler) Create(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CreateTutorialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	tut, err := h.svc.CreateTutorial(c.Request.Context(), ownerSub, service.CreateTutorialParams{
		Title:       req.Title,
		Subject:     req.Subject,
		Description: req.Description,
		Difficulty:  req.Difficulty,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toTutorialResponse(*tut))
}

// Get godoc
//
//	@Summary  Get a tutorial
//	@Tags     tutorials
//	@Produce  json
//	@Param    id   path      string  true  "Tutorial ID"
//	@Success  200  {object}  apphttp.TutorialResponse
//	@Router   /v1/tutorials/{id} [get]
func (h *TutorialHandler) Get(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	tut, err := h.svc.GetTutorial(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTutorialResponse(*tut))
}

// Update godoc
//
//	@Summary  Partially update a tutorial
//	@Tags     tutorials
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                         true  "Tutorial ID"
//	@Param    body  body      apphttp.UpdateTutorialRequest  true  "fields to update"
//	@Success  200   {object}  apphttp.TutorialResponse
//	@Router   /v1/tutorials/{id} [patch]
func (h *TutorialHandler) Update(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.UpdateTutorialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	tut, err := h.svc.UpdateTutorial(c.Request.Context(), c.Param("id"), ownerSub, service.UpdateTutorialParams{
		Title:       req.Title,
		Subject:     req.Subject,
		Description: req.Description,
		Difficulty:  req.Difficulty,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTutorialResponse(*tut))
}

// Delete godoc
//
//	@Summary  Delete a tutorial
//	@Tags     tutorials
//	@Param    id  path  string  true  "Tutorial ID"
//	@Success  204
//	@Router   /v1/tutorials/{id} [delete]
func (h *TutorialHandler) Delete(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	if err := h.svc.DeleteTutorial(c.Request.Context(), c.Param("id"), ownerSub); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Session sub-routes ─────────────────────────────────────────────────────────

// CreateSession godoc
//
//	@Summary  Create a session for a tutorial
//	@Tags     tutorial-sessions
//	@Accept   json
//	@Produce  json
//	@Param    id   path      string                              true   "Tutorial ID"
//	@Param    body body      apphttp.CreateTutorialSessionRequest false  "Session parameters"
//	@Success  201  {object}  apphttp.TutorialSessionResponse
//	@Router   /v1/tutorials/{id}/sessions [post]
func (h *TutorialHandler) CreateSession(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CreateTutorialSessionRequest
	// Accept empty body for backward compatibility
	_ = c.ShouldBindJSON(&req)

	sess, err := h.sessionSvc.CreateTutorialSession(
		c.Request.Context(),
		ownerSub,
		c.Param("id"),
		domain.TutorialSessionKind(req.Kind),
	)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toTutorialSessionResponse(*sess))
}

// ListSessions godoc
//
//	@Summary  List sessions for a tutorial
//	@Tags     tutorial-sessions
//	@Produce  json
//	@Param    id   path      string  true  "Tutorial ID"
//	@Success  200  {array}   apphttp.TutorialSessionResponse
//	@Router   /v1/tutorials/{id}/sessions [get]
func (h *TutorialHandler) ListSessions(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sessions, err := h.sessionSvc.ListTutorialSessions(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	resp := make([]apphttp.TutorialSessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = toTutorialSessionResponse(s)
	}
	c.JSON(http.StatusOK, resp)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func toTutorialResponse(t domain.Tutorial) apphttp.TutorialResponse {
	return apphttp.TutorialResponse{
		ID:          t.ID,
		Title:       t.Title,
		Subject:     t.Subject,
		Description: t.Description,
		Difficulty:  t.Difficulty,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toTutorialSessionResponse(s domain.TutorialSession) apphttp.TutorialSessionResponse {
	return apphttp.TutorialSessionResponse{
		ID:         s.ID,
		TutorialID: s.TutorialID,
		Status:     s.Status,
		Kind:       s.Kind,
		Notes:      s.Notes,
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
	}
}

func toArtifactResponse(a domain.Artifact) apphttp.ArtifactResponse {
	return apphttp.ArtifactResponse{
		ID:           a.ID,
		SessionID:    a.SessionID,
		Kind:         a.Kind,
		Title:        a.Title,
		Content:      a.Content,
		ProblemSetID: a.ProblemSetID,
		CreatedAt:    a.CreatedAt,
	}
}

func toTutorialTurnResponse(t domain.TutorialTurn) apphttp.TutorialTurnResponse {
	return apphttp.TutorialTurnResponse{
		ID:        t.ID,
		SessionID: t.SessionID,
		Speaker:   t.Speaker,
		Text:      t.Text,
		CreatedAt: t.CreatedAt,
	}
}

func toProblemSetResponse(ps domain.ProblemSet) apphttp.ProblemSetResponse {
	tasks := make([]apphttp.ProblemSetTaskResponse, len(ps.Tasks))
	for i, t := range ps.Tasks {
		tasks[i] = apphttp.ProblemSetTaskResponse{
			PatternCode: string(t.PatternCode),
			Title:       t.Title,
			Description: t.Description,
			Prompt:      t.Prompt,
		}
	}
	return apphttp.ProblemSetResponse{
		ID:                    ps.ID,
		TutorialID:            ps.TutorialID,
		WeekOf:                ps.WeekOf,
		AssignedFromSessionID: ps.AssignedFromSessionID,
		Status:                ps.Status,
		Tasks:                 tasks,
		ReviewNotes:           ps.ReviewNotes,
		CreatedAt:             ps.CreatedAt,
		UpdatedAt:             ps.UpdatedAt,
	}
}

// handleServiceError maps well-known service errors to HTTP status codes.
func handleServiceError(c *gin.Context, err error) {
	var notFound *service.NotFoundError
	if errors.As(err, &notFound) {
		apphttp.Fail(c, http.StatusNotFound, "not_found", err.Error())
		return
	}

	var valErr *service.ValidationError
	if errors.As(err, &valErr) {
		apphttp.FailDetails(c, http.StatusBadRequest, "validation_error", err.Error(), gin.H{
			"field":   valErr.Field,
			"message": valErr.Message,
		})
		return
	}

	// Log the actual error for debugging
	_ = c.Error(err)
	apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}
