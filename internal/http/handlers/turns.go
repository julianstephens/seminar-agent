package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/julianstephens/formation/internal/auth"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/referee"
	"github.com/julianstephens/formation/internal/service"
)

// TurnHandler exposes the turn-submission route.
type TurnHandler struct {
	svc *service.TurnService
}

// NewTurnHandler constructs a TurnHandler backed by the given service.
func NewTurnHandler(svc *service.TurnService) *TurnHandler {
	return &TurnHandler{svc: svc}
}

// Register wires turn routes onto the provided router group.
// Expected prefix: /v1/sessions  (turns are scoped to a session via :id).
func (h *TurnHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/:id/turns", h.SubmitTurn)
}

// SubmitTurn godoc
//
//	@Summary  Submit a user turn and receive the agent response
//	@Tags     turns
//	@Accept   json
//	@Produce  json
//	@Param    id    path      string                       true  "Session ID"
//	@Param    body  body      apphttp.SubmitTurnRequest    true  "turn text"
//	@Success  201   {object}  apphttp.SubmitTurnResponse
//	@Router   /v1/sessions/{id}/turns [post]
func (h *TurnHandler) SubmitTurn(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		return
	}

	var req apphttp.SubmitTurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apphttp.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.svc.SubmitTurn(c.Request.Context(), c.Param("id"), ownerSub,
		service.SubmitTurnParams{Text: req.Text},
	)
	if err != nil {
		handleTurnServiceError(c, err)
		return
	}

	// Safety check: result should never be nil when err is nil, but be defensive
	if result == nil || result.UserTurn == nil {
		apphttp.Fail(c, http.StatusInternalServerError, "internal_error",
			"invalid service response: missing user turn")
		return
	}

	resp := apphttp.SubmitTurnResponse{
		UserTurn: toTurnResponse(*result.UserTurn),
	}
	if result.AgentTurn != nil {
		ar := toTurnResponse(*result.AgentTurn)
		resp.AgentTurn = &ar
	}

	c.JSON(http.StatusCreated, resp)
}

// handleTurnServiceError extends handleSessionServiceError with the
// missing_locator error produced by the referee.
func handleTurnServiceError(c *gin.Context, err error) {
	var missingLocator *referee.ErrMissingLocator
	if errors.As(err, &missingLocator) {
		apphttp.Fail(c, http.StatusBadRequest, "missing_locator", err.Error())
		return
	}
	handleSessionServiceError(c, err)
}
