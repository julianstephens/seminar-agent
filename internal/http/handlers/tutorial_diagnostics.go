package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/julianstephens/formation/internal/auth"
	"github.com/julianstephens/formation/internal/domain"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/repo"
	"github.com/julianstephens/formation/internal/service"
)

// TutorialDiagnosticsHandler handles HTTP requests for tutorial diagnostic data.
type TutorialDiagnosticsHandler struct {
	diagnosticSvc *service.DiagnosticLedgerService
	repo          *repo.TutorialRepo
}

// NewTutorialDiagnosticsHandler constructs a handler backed by the given service.
func NewTutorialDiagnosticsHandler(svc *service.DiagnosticLedgerService, repo *repo.TutorialRepo) *TutorialDiagnosticsHandler {
	return &TutorialDiagnosticsHandler{
		diagnosticSvc: svc,
		repo:          repo,
	}
}

// Register mounts the diagnostic routes onto the given router group.
func (h *TutorialDiagnosticsHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id/diagnostics", h.ListDiagnostics)
	rg.GET("/:id/diagnostics/summary", h.GetDiagnosticSummary)
	rg.GET("/:id/problem-sets", h.ListProblemSets)
}

// ListDiagnostics godoc
//
//	@Summary  List diagnostic entries for a tutorial
//	@Tags     tutorials
//	@Param    id  path  string  true  "tutorial ID"
//	@Success  200  {object}  apphttp.ListDiagnosticsResponse
//	@Failure  404  {object}  apphttp.ErrorResponse
//	@Failure  500  {object}  apphttp.ErrorResponse
//	@Router   /v1/tutorials/{id}/diagnostics [get]
func (h *TutorialDiagnosticsHandler) ListDiagnostics(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}
	tutorialID := c.Param("id")

	entries, err := h.repo.ListDiagnosticEntriesByTutorial(
		c.Request.Context(),
		tutorialID,
		ownerSub,
	)
	if err != nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "failed to list diagnostics")
		return
	}

	resp := apphttp.ListDiagnosticsResponse{
		Entries: toDiagnosticEntryDTOs(entries),
	}
	c.JSON(http.StatusOK, resp)
}

// GetDiagnosticSummary godoc
//
//	@Summary  Get pattern summary for a tutorial
//	@Tags     tutorials
//	@Param    id  path   string  true   "tutorial ID"
//	@Param    weeks  query  int     false  "number of weeks to look back (default: 4)"
//	@Success  200  {object}  apphttp.DiagnosticSummaryResponse
//	@Failure  404  {object}  apphttp.ErrorResponse
//	@Failure  500  {object}  apphttp.ErrorResponse
//	@Router   /v1/tutorials/{id}/diagnostics/summary [get]
func (h *TutorialDiagnosticsHandler) GetDiagnosticSummary(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}
	tutorialID := c.Param("id")

	weeks := 4
	if w := c.Query("weeks"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			weeks = parsed
		}
	}

	summary, err := h.diagnosticSvc.BuildPatternSummary(
		c.Request.Context(),
		tutorialID,
		ownerSub,
		weeks,
	)
	if err != nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "failed to build pattern summary")
		return
	}

	resp := apphttp.DiagnosticSummaryResponse{
		Items: toPatternSummaryItemDTOs(summary.Items),
	}
	c.JSON(http.StatusOK, resp)
}

// ListProblemSets godoc
//
//	@Summary  List problem sets for a tutorial
//	@Tags     tutorials
//	@Param    id  path  string  true  "tutorial ID"
//	@Success  200  {object}  apphttp.ListProblemSetsResponse
//	@Failure  404  {object}  apphttp.ErrorResponse
//	@Failure  500  {object}  apphttp.ErrorResponse
//	@Router   /v1/tutorials/{id}/problem-sets [get]
func (h *TutorialDiagnosticsHandler) ListProblemSets(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}
	tutorialID := c.Param("id")

	problemSets, err := h.repo.ListProblemSets(
		c.Request.Context(),
		tutorialID,
		ownerSub,
	)
	if err != nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error", "failed to list problem sets")
		return
	}

	resp := apphttp.ListProblemSetsResponse{
		ProblemSets: toProblemSetDTOs(problemSets),
	}
	c.JSON(http.StatusOK, resp)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toDiagnosticEntryDTOs(entries []domain.DiagnosticEntry) []apphttp.DiagnosticEntryResponse {
	result := make([]apphttp.DiagnosticEntryResponse, len(entries))
	for i, e := range entries {
		result[i] = apphttp.DiagnosticEntryResponse{
			ID:                e.ID,
			TutorialID:        e.TutorialID,
			TutorialSessionID: e.TutorialSessionID,
			WeekOf:            e.WeekOf,
			PatternCode:       string(e.PatternCode),
			Severity:          e.Severity,
			Status:            string(e.Status),
			Evidence:          e.Evidence,
			Notes:             e.Notes,
			CreatedAt:         e.CreatedAt,
			UpdatedAt:         e.UpdatedAt,
		}
	}
	return result
}

func toPatternSummaryItemDTOs(items []service.PatternSummaryItem) []apphttp.PatternSummaryItemResponse {
	result := make([]apphttp.PatternSummaryItemResponse, len(items))
	for i, item := range items {
		result[i] = apphttp.PatternSummaryItemResponse{
			PatternCode:  item.PatternCode,
			Occurrences:  item.Occurrences,
			LastSeenWeek: item.LastSeenWeek,
			Trend:        item.Trend,
		}
	}
	return result
}

func toProblemSetDTOs(problemSets []domain.ProblemSet) []apphttp.ProblemSetResponse {
	result := make([]apphttp.ProblemSetResponse, len(problemSets))
	for i, ps := range problemSets {
		result[i] = toProblemSetDTO(ps)
	}
	return result
}

func toProblemSetDTO(ps domain.ProblemSet) apphttp.ProblemSetResponse {
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
