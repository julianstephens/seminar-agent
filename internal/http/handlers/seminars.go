// Package handlers contains the HTTP handler implementations for the v1 API.
// Each handler delegates to a service, translates service errors to HTTP
// status codes, and writes the standard response envelope.
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

// SeminarHandler exposes seminar-related routes.
type SeminarHandler struct {
	svc *service.SeminarService
}

// NewSeminarHandler constructs a SeminarHandler backed by the given service.
func NewSeminarHandler(svc *service.SeminarService) *SeminarHandler {
	return &SeminarHandler{svc: svc}
}

// ── Route registration ─────────────────────────────────────────────────────────

// Register wires all seminar routes onto the provided router group.
// The group is expected to already have the JWT middleware applied.
func (h *SeminarHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:id", h.Get)
	rg.PATCH("/:id", h.Update)
	rg.DELETE("/:id", h.Delete)
}

// ── Handlers ───────────────────────────────────────────────────────────────────

// List godoc
//
//	@Summary  List seminars
//	@Tags     seminars
//	@Produce  json
//	@Success  200  {array}   apphttp.SeminarResponse
//	@Router   /v1/seminars   [get]
func (h *SeminarHandler) List(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	seminars, err := h.svc.List(c.Request.Context(), ownerSub)
	if err != nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "failed to list seminars")
		return
	}

	resp := make([]apphttp.SeminarResponse, len(seminars))
	for i, s := range seminars {
		resp[i] = toSeminarResponse(s)
	}
	c.JSON(http.StatusOK, resp)
}

// Create godoc
//
//	@Summary  Create a seminar
//	@Tags     seminars
//	@Accept   json
//	@Produce  json
//	@Param    body  body      apphttp.CreateSeminarRequest  true  "seminar fields"
//	@Success  201   {object}  apphttp.SeminarResponse
//	@Router   /v1/seminars [post]
func (h *SeminarHandler) Create(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.CreateSeminarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	sem, err := h.svc.Create(c.Request.Context(), ownerSub, service.CreateParams{
		Title:               req.Title,
		Author:              req.Author,
		EditionNotes:        req.EditionNotes,
		ThesisCurrent:       req.ThesisCurrent,
		DefaultMode:         req.DefaultMode,
		DefaultReconMinutes: req.DefaultReconMinutes,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toSeminarResponse(*sem))
}

// Get godoc
//
//	@Summary  Get a seminar
//	@Tags     seminars
//	@Produce  json
//	@Param    id   path      string  true  "Seminar ID"
//	@Success  200  {object}  apphttp.SeminarResponse
//	@Router   /v1/seminars/{id} [get]
func (h *SeminarHandler) Get(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	sem, err := h.svc.Get(c.Request.Context(), c.Param("id"), ownerSub)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSeminarResponse(*sem))
}

// Update godoc
//
//	@Summary  Partially update a seminar
//	@Tags     seminars
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                        true  "Seminar ID"
//	@Param    body  body      apphttp.UpdateSeminarRequest  true  "fields to update"
//	@Success  200   {object}  apphttp.SeminarResponse
//	@Router   /v1/seminars/{id} [patch]
func (h *SeminarHandler) Update(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.UpdateSeminarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	sem, err := h.svc.Update(c.Request.Context(), c.Param("id"), ownerSub, service.UpdateParams{
		Title:               req.Title,
		Author:              req.Author,
		EditionNotes:        req.EditionNotes,
		DefaultMode:         req.DefaultMode,
		DefaultReconMinutes: req.DefaultReconMinutes,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSeminarResponse(*sem))
}

// Delete godoc
//
//	@Summary  Delete a seminar
//	@Tags     seminars
//	@Param    id  path  string  true  "Seminar ID"
//	@Success  204
//	@Router   /v1/seminars/{id} [delete]
func (h *SeminarHandler) Delete(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), c.Param("id"), ownerSub); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// toSeminarResponse converts a domain.Seminar into its HTTP response shape.
func toSeminarResponse(s domain.Seminar) apphttp.SeminarResponse {
	return apphttp.SeminarResponse{
		ID:                  s.ID,
		Title:               s.Title,
		Author:              s.Author,
		EditionNotes:        s.EditionNotes,
		ThesisCurrent:       s.ThesisCurrent,
		DefaultMode:         s.DefaultMode,
		DefaultReconMinutes: s.DefaultReconMinutes,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
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

	apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}
