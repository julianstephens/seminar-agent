package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/julianstephens/formation/internal/auth"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/modules/tutorial/service"
	"github.com/julianstephens/formation/internal/observability"
	"github.com/julianstephens/formation/internal/sse"
)

// TutorialSessionEventsHandler streams server-sent events for tutorial sessions.
type TutorialSessionEventsHandler struct {
	hub      *sse.Hub
	sessions *service.TutorialSessionService
}

// NewTutorialSessionEventsHandler constructs a TutorialSessionEventsHandler.
func NewTutorialSessionEventsHandler(
	hub *sse.Hub,
	sessions *service.TutorialSessionService,
) *TutorialSessionEventsHandler {
	return &TutorialSessionEventsHandler{hub: hub, sessions: sessions}
}

// Register wires the SSE endpoint onto the provided router group.
// Expected prefix: /v1/tutorial-sessions
func (h *TutorialSessionEventsHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id/events", h.Stream)
}

// Stream godoc
//
//	@Summary  Open SSE stream for a tutorial session
//	@Tags     tutorial-sessions
//	@Produce  text/event-stream
//	@Param    id   path  string  true  "Session ID"
//	@Success  200
//	@Router   /v1/tutorial-sessions/{id}/events [get]
func (h *TutorialSessionEventsHandler) Stream(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		// MustOwnerSub has already written 401
		return
	}

	sessionID := c.Param("id")

	// Verify the session exists and belongs to this user.
	_, err = h.sessions.GetTutorialSession(c.Request.Context(), sessionID, ownerSub)
	if err != nil {
		apphttp.Fail(c, http.StatusNotFound, "not_found", fmt.Sprintf("tutorial session %s not found", sessionID))
		return
	}

	// Negotiate SSE.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	// Subscribe to the hub before flushing headers so no events are missed.
	events, unsub := h.hub.Subscribe(sessionID, ownerSub)
	defer unsub()

	logger := observability.LoggerFromContext(c.Request.Context())
	logger.Info("sse: connection established",
		"session_id", sessionID,
		"owner", ownerSub,
	)

	// Flush headers immediately so the browser receives the streaming response.
	flusher, hasFlusher := c.Writer.(http.Flusher)
	if !hasFlusher {
		apphttp.Fail(c, http.StatusInternalServerError, "no_flusher", "streaming not supported")
		return
	}
	c.Status(http.StatusOK)
	flusher.Flush()

	heartbeatTicker := time.NewTicker(sse.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	w := c.Writer

	write := func(raw string) bool {
		_, writeErr := fmt.Fprint(w, raw)
		if writeErr != nil {
			logger.Warn("sse: write error",
				"session_id", sessionID,
				"error", writeErr.Error(),
			)
			return false
		}
		flusher.Flush()
		return true
	}

	sendEvent := func(e sse.Event) bool {
		raw, fmtErr := sse.Format(e)
		if fmtErr != nil {
			logger.Warn("sse: failed to format event",
				"session_id", sessionID,
				"event_type", e.Type,
				"error", fmtErr.Error(),
			)
			return true // skip malformed; don't close the stream
		}
		logger.Debug("sse: sending event",
			"session_id", sessionID,
			"event_type", e.Type,
		)
		return write(raw)
	}

	for {
		select {
		// Client disconnected.
		case <-c.Request.Context().Done():
			logger.Info("sse: client disconnected",
				"session_id", sessionID,
			)
			return

		// Heartbeat comment to keep the connection alive through proxies.
		case <-heartbeatTicker.C:
			if !write(sse.Heartbeat()) {
				return
			}

		// Hub-broadcast event.
		case ev, ok := <-events:
			if !ok {
				// Hub closed the channel (subscriber removed).
				return
			}
			if !sendEvent(ev) {
				return
			}
		}
	}
}
