// Package sse provides the server-sent event hub that delivers realtime session
// events to connected browser clients.
//
// Architecture
// ────────────
// The Hub maintains a map of sessionID → []subscriber. Each subscriber is a
// buffered channel of Events. When a new event is published for a session every
// open subscriber channel receives a copy; slow or stalled consumers are dropped
// rather than blocking the caller.
//
// A timer_tick event is generated per-subscriber inside the EventsHandler rather
// than inside the Hub so that the tick frequency places no coordination burden on
// the Hub itself.
//
// Event types emitted:
//
//	phase_changed      – scheduler advanced the phase (data: PhaseChangedPayload)
//	timer_tick         – 1-second countdown heartbeat  (data: TimerTickPayload)
//	turn_added         – a new turn was persisted       (data: TurnAddedPayload)
//	session_completed  – session reached done/complete  (data: SessionCompletedPayload)
//	error              – non-fatal stream error         (data: ErrorPayload)
package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/julianstephens/formation/internal/domain"
)

// ── Event types ───────────────────────────────────────────────────────────────

const (
	EventPhaseChanged       = "phase_changed"
	EventTimerTick          = "timer_tick"
	EventTurnAdded          = "turn_added"
	EventAgentResponseChunk = "agent_response_chunk"
	EventSessionCompleted   = "session_completed"
	EventError              = "error"
)

// Event is a single SSE message.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// PhaseChangedPayload is carried by phase_changed events.
type PhaseChangedPayload struct {
	SessionID   string    `json:"session_id"`
	Phase       string    `json:"phase"`
	PhaseEndsAt time.Time `json:"phase_ends_at,omitempty"`
}

// TimerTickPayload is carried by timer_tick events.
type TimerTickPayload struct {
	SessionID        string  `json:"session_id"`
	Phase            string  `json:"phase"`
	SecondsRemaining float64 `json:"seconds_remaining"`
}

// TurnAddedPayload is carried by turn_added events.
type TurnAddedPayload struct {
	SessionID string      `json:"session_id"`
	Turn      domain.Turn `json:"turn"`
}

// TutorialTurnAddedPayload is carried by turn_added events for tutorial sessions.
type TutorialTurnAddedPayload struct {
	SessionID string              `json:"session_id"`
	Turn      domain.TutorialTurn `json:"turn"`
}

// AgentResponseChunkPayload is carried by agent_response_chunk events.
type AgentResponseChunkPayload struct {
	SessionID string `json:"session_id"`
	TurnID    string `json:"turn_id"`
	Chunk     string `json:"chunk"`
	IsFinal   bool   `json:"is_final"`
}

// SessionCompletedPayload is carried by session_completed events.
type SessionCompletedPayload struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// ErrorPayload is carried by error events.
type ErrorPayload struct {
	Message string `json:"message"`
}

// ── Hub ───────────────────────────────────────────────────────────────────────

const (
	// subBufSize is the per-subscriber channel capacity. Events beyond this
	// capacity are dropped rather than blocking the publisher.
	subBufSize = 32

	// Heartbeat comment interval keeps connections alive through proxies.
	HeartbeatInterval = 15 * time.Second
)

type subscriber struct {
	ch       chan Event
	ownerSub string
}

// Hub is the central SSE broker.
// All methods are safe for concurrent use.
type Hub struct {
	mu   sync.RWMutex
	subs map[string][]*subscriber // session_id → subscribers
	log  *slog.Logger
}

// New creates a ready-to-use Hub.
func New(logger *slog.Logger) *Hub {
	return &Hub{
		subs: make(map[string][]*subscriber),
		log:  logger,
	}
}

// Subscribe registers a new SSE subscriber for sessionID and returns:
//   - a receive-only channel that delivers events,
//   - an unsubscribe function the caller MUST invoke when the connection closes.
//
// The ownerSub argument is used only for log attribution.
func (h *Hub) Subscribe(sessionID, ownerSub string) (<-chan Event, func()) {
	sub := &subscriber{
		ch:       make(chan Event, subBufSize),
		ownerSub: ownerSub,
	}

	h.mu.Lock()
	h.subs[sessionID] = append(h.subs[sessionID], sub)
	h.mu.Unlock()

	h.log.Debug("sse: subscriber added",
		"session", sessionID,
		"owner", ownerSub,
	)

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		list := h.subs[sessionID]
		updated := list[:0]
		for _, s := range list {
			if s != sub {
				updated = append(updated, s)
			}
		}
		if len(updated) == 0 {
			delete(h.subs, sessionID)
		} else {
			h.subs[sessionID] = updated
		}
		close(sub.ch)

		h.log.Debug("sse: subscriber removed",
			"session", sessionID,
			"owner", ownerSub,
		)
	}

	return sub.ch, unsub
}

// Publish delivers event to every subscriber of sessionID.
// Subscribers whose buffer is full are silently dropped to avoid head-of-line
// blocking. This method never blocks.
func (h *Hub) Publish(sessionID string, event Event) {
	h.mu.RLock()
	list := h.subs[sessionID]
	h.mu.RUnlock()

	dropped := 0
	for _, s := range list {
		select {
		case s.ch <- event:
		default:
			dropped++
		}
	}

	if dropped > 0 {
		h.log.Warn("sse: dropped events for slow subscribers",
			"session", sessionID,
			"event", event.Type,
			"dropped", dropped,
		)
	}
}

// PublishPhaseChanged emits a phase_changed event for sessionID based on the
// updated session returned by the scheduler's AdvancePhase call.
func (h *Hub) PublishPhaseChanged(sess *domain.Session) {
	payload := PhaseChangedPayload{
		SessionID:   sess.ID,
		Phase:       string(sess.Phase),
		PhaseEndsAt: sess.PhaseEndsAt,
	}
	h.Publish(sess.ID, Event{Type: EventPhaseChanged, Data: payload})

	// If the session just completed, also emit a session_completed event.
	if sess.IsTerminal() {
		h.PublishSessionCompleted(sess.ID, string(sess.Status))
	}
}

// PublishTurnAdded emits a turn_added event for the given turn.
func (h *Hub) PublishTurnAdded(t *domain.Turn) {
	h.Publish(t.SessionID, Event{
		Type: EventTurnAdded,
		Data: TurnAddedPayload{SessionID: t.SessionID, Turn: *t},
	})
}

// PublishTutorialTurnAdded emits a turn_added event for the given tutorial turn.
func (h *Hub) PublishTutorialTurnAdded(t *domain.TutorialTurn) {
	h.Publish(t.SessionID, Event{
		Type: EventTurnAdded,
		Data: TutorialTurnAddedPayload{SessionID: t.SessionID, Turn: *t},
	})
}

// PublishAgentResponseChunk emits an agent_response_chunk event for streaming agent responses.
func (h *Hub) PublishAgentResponseChunk(sessionID, turnID, chunk string, isFinal bool) {
	h.log.Debug("sse: publishing chunk",
		"session", sessionID,
		"turn", turnID,
		"chunk_len", len(chunk),
		"is_final", isFinal,
	)
	h.Publish(sessionID, Event{
		Type: EventAgentResponseChunk,
		Data: AgentResponseChunkPayload{
			SessionID: sessionID,
			TurnID:    turnID,
			Chunk:     chunk,
			IsFinal:   isFinal,
		},
	})
}

// PublishSessionCompleted emits a session_completed event.
func (h *Hub) PublishSessionCompleted(sessionID, status string) {
	h.Publish(sessionID, Event{
		Type: EventSessionCompleted,
		Data: SessionCompletedPayload{SessionID: sessionID, Status: status},
	})
}

// PublishError emits an error event to all subscribers of sessionID.
func (h *Hub) PublishError(sessionID, message string) {
	h.Publish(sessionID, Event{
		Type: EventError,
		Data: ErrorPayload{Message: message},
	})
}

// ── SSE wire format ───────────────────────────────────────────────────────────

// Format serialises an Event to the SSE wire format:
//
//	event: <type>\n
//	data: <json>\n\n
func Format(e Event) (string, error) {
	b, err := json.Marshal(e.Data)
	if err != nil {
		return "", fmt.Errorf("sse marshal: %w", err)
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", e.Type, b), nil
}

// Heartbeat returns an SSE comment line that keeps the connection alive.
func Heartbeat() string {
	return ": heartbeat\n\n"
}
