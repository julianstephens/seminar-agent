package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/julianstephens/formation/internal/auth"
	apphttp "github.com/julianstephens/formation/internal/http"
	"github.com/julianstephens/formation/internal/modules/seminar/service"
	"github.com/julianstephens/formation/internal/sse"
)

// EventsHandler streams server-sent events for a single session.
type EventsHandler struct {
	hub      *sse.Hub
	sessions *service.SeminarSessionService
}

// NewEventsHandler constructs an EventsHandler.
func NewEventsHandler(hub *sse.Hub, sessions *service.SeminarSessionService) *EventsHandler {
	return &EventsHandler{hub: hub, sessions: sessions}
}

// Register wires the SSE endpoint onto the provided router group.
// Expected prefix: /v1/sessions
func (h *EventsHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/:id/events", h.Stream)
}

// Stream godoc
//
//	@Summary  Open SSE stream for a session
//	@Tags     sessions
//	@Produce  text/event-stream
//	@Param    id   path  string  true  "Session ID"
//	@Success  200
//	@Router   /v1/sessions/{id}/events [get]
func (h *EventsHandler) Stream(c *gin.Context) {
	ownerSub, err := auth.MustOwnerSub(c)
	if err != nil {
		// MustOwnerSub has already written 401
		return
	}

	sessionID := c.Param("id")

	// Verify the session exists and belongs to this user.
	detail, err := h.sessions.Get(c.Request.Context(), sessionID, ownerSub)
	if err != nil {
		apphttp.Fail(c, http.StatusNotFound, "not_found", fmt.Sprintf("session %s not found", sessionID))
		return
	}

	// Negotiate SSE.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	// Subscribe to the hub before flushing headers so no events are missed.
	events, unsub, err := h.hub.Subscribe(sessionID, ownerSub)
	if err != nil {
		if errors.Is(err, sse.ErrTooManySubscribers) {
			apphttp.Fail(c, http.StatusTooManyRequests, "too_many_subscribers",
				"Too many concurrent connections for this session")
		} else {
			apphttp.Fail(c, http.StatusInternalServerError, "subscribe_error", err.Error())
		}
		return
	}
	defer unsub()

	// Flush headers immediately so the browser receives the streaming response.
	flusher, hasFlusher := c.Writer.(http.Flusher)
	if !hasFlusher {
		apphttp.Fail(
			c,
			http.StatusInternalServerError,
			"no_flusher",
			"Server-sent events are not supported by this server",
		)
		return
	}
	flusher.Flush()

	heartbeatTicker := time.NewTicker(sse.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	tickTicker := time.NewTicker(time.Second)
	defer tickTicker.Stop()

	// Cache the session for timer_tick computation; refresh on phase_changed.
	sess := detail.Session

	w := c.Writer

	write := func(raw string) bool {
		_, writeErr := fmt.Fprint(w, raw)
		if writeErr != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	sendEvent := func(e sse.Event) bool {
		raw, fmtErr := sse.Format(e)
		if fmtErr != nil {
			return true // skip malformed; don't close the stream
		}
		return write(raw)
	}

	for {
		select {
		// Client disconnected.
		case <-c.Request.Context().Done():
			return

		// Heartbeat comment to keep the connection alive through proxies.
		case <-heartbeatTicker.C:
			if !write(sse.Heartbeat()) {
				return
			}

		// Per-second timer tick – only for timed phases.
		case <-tickTicker.C:
			if !sess.IsTerminal() && !sess.PhaseEndsAt.IsZero() {
				remaining := time.Until(sess.PhaseEndsAt).Seconds()
				if remaining < 0 {
					remaining = 0
				}
				e := sse.Event{
					Type: sse.EventTimerTick,
					Data: sse.TimerTickPayload{
						SessionID:        sessionID,
						Phase:            string(sess.Phase),
						SecondsRemaining: remaining,
					},
				}
				if !sendEvent(e) {
					return
				}
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
			// Refresh cached session on phase changes so tick math stays correct.
			if ev.Type == sse.EventPhaseChanged {
				if updated, getErr := h.sessions.Get(c.Request.Context(), sessionID, ownerSub); getErr == nil {
					sess = updated.Session
				}
			}
		}
	}
}
