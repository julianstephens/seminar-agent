// Package sse provides the server-sent event hub that delivers realtime session
// events to connected browser clients.
//
// Architecture
// ────────────
// The Hub maintains a map of sessionID → []subscriber. Each subscriber is a
// buffered channel of Events. When a new event is published for a session it is
// pushed to Redis Pub/Sub; a per-session background goroutine subscribes to the
// Redis channel and fans out to every local subscriber channel. This allows
// multiple application instances to share events (horizontal scaling).
//
// Slow consumers are detected by non-blocking channel sends. After
// maxConsecutiveDrops failed sends, or after a subscriber's buffer has been
// full for idleTimeout, the subscriber is automatically evicted.
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
//	agent_response_chunk – streaming LLM token chunk   (data: AgentResponseChunkPayload)
//	session_completed  – session reached done/complete  (data: SessionCompletedPayload)
//	error              – non-fatal stream error         (data: ErrorPayload)
package sse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"

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

// ErrTooManySubscribers is returned by Subscribe when the per-session limit is reached.
var ErrTooManySubscribers = errors.New("sse: too many subscribers for this session")

// Event is a single SSE message.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// PhaseChangedPayload is carried by phase_changed events.
type PhaseChangedPayload struct {
	SessionID      string    `json:"session_id"`
	Phase          string    `json:"phase"`
	PhaseEndsAt    time.Time `json:"phase_ends_at,omitempty"`
	PhaseStartedAt time.Time `json:"phase_started_at,omitempty"`
}

// TimerTickPayload is carried by timer_tick events.
type TimerTickPayload struct {
	SessionID        string  `json:"session_id"`
	Phase            string  `json:"phase"`
	SecondsRemaining float64 `json:"seconds_remaining"`
}

// TurnAddedPayload is carried by turn_added events.
type TurnAddedPayload struct {
	SessionID string             `json:"session_id"`
	Turn      domain.SeminarTurn `json:"turn"`
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
	// subBufSize is the per-subscriber channel capacity.
	// 256 accommodates high-frequency token streaming (~100 tokens/s bursts).
	subBufSize = 256

	// maxConsecutiveDrops is the number of consecutive failed sends before a
	// subscriber is automatically evicted.
	maxConsecutiveDrops = 3

	// maxSubscribers is the maximum number of concurrent subscribers per session.
	maxSubscribers = 100

	// idleTimeout is the maximum duration a subscriber may have a full buffer
	// before being evicted, regardless of drop count.
	idleTimeout = 30 * time.Second

	// HeartbeatInterval keeps connections alive through proxies.
	HeartbeatInterval = 15 * time.Second
)

// ── Prometheus metrics ────────────────────────────────────────────────────────

var (
	metricSubscribers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sse_subscribers_current",
		Help: "Current number of active SSE subscriber connections.",
	})
	metricDroppedEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sse_dropped_events_total",
		Help: "Total SSE events dropped because a subscriber channel was full.",
	})
	metricEvictedSubscribers = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sse_evicted_subscribers_total",
		Help: "Total SSE subscribers evicted due to slow consumption or idle timeout.",
	})
)

// ── subscriber ────────────────────────────────────────────────────────────────

type subscriber struct {
	ch          chan Event
	ownerSub    string
	drops       atomic.Int32 // consecutive failed channel sends
	firstDropAt atomic.Int64 // UnixNano of first consecutive drop; 0 = not dropping
	closeOnce   sync.Once    // ensures the channel is closed exactly once
}

func (s *subscriber) safeClose() {
	s.closeOnce.Do(func() { close(s.ch) })
}

// ── Hub ───────────────────────────────────────────────────────────────────────

// Hub is the central SSE broker backed by Redis Pub/Sub for horizontal scaling.
// All methods are safe for concurrent use.
type Hub struct {
	mu   sync.RWMutex
	subs map[string][]*subscriber // session_id → local subscribers

	// Redis Pub/Sub management – one background goroutine per session with
	// at least one local subscriber.
	sessionsMu sync.Mutex
	sessions   map[string]context.CancelFunc // session_id → cancel fn

	log      *slog.Logger
	redis    *redis.Client
	redisPfx string
}

// New creates a ready-to-use Hub backed by rdb for cross-instance delivery.
// prefix is prepended to every Redis key (e.g. "formation:").
func New(logger *slog.Logger, rdb *redis.Client, prefix string) *Hub {
	return &Hub{
		subs:     make(map[string][]*subscriber),
		sessions: make(map[string]context.CancelFunc),
		log:      logger,
		redis:    rdb,
		redisPfx: prefix,
	}
}

func (h *Hub) channelName(sessionID string) string {
	return h.redisPfx + "sse:" + sessionID
}

// Subscribe registers a new SSE subscriber for sessionID and returns:
//   - a receive-only channel that delivers events,
//   - an unsubscribe function the caller MUST invoke when the connection closes,
//   - ErrTooManySubscribers when the per-session limit (maxSubscribers) is reached.
//
// The ownerSub argument is used only for log attribution.
func (h *Hub) Subscribe(sessionID, ownerSub string) (<-chan Event, func(), error) {
	sub := &subscriber{
		ch:       make(chan Event, subBufSize),
		ownerSub: ownerSub,
	}

	h.mu.Lock()
	if len(h.subs[sessionID]) >= maxSubscribers {
		h.mu.Unlock()
		return nil, nil, ErrTooManySubscribers
	}
	h.subs[sessionID] = append(h.subs[sessionID], sub)
	isFirst := len(h.subs[sessionID]) == 1
	h.mu.Unlock()

	metricSubscribers.Inc()

	// If this is the first local subscriber for this session, start the Redis
	// subscription goroutine so that events from remote instances are delivered.
	if isFirst {
		h.sessionsMu.Lock()
		if _, exists := h.sessions[sessionID]; !exists {
			ctx, cancel := context.WithCancel(context.Background())
			h.sessions[sessionID] = cancel
			go h.runRedisSubscription(ctx, sessionID)
		}
		h.sessionsMu.Unlock()
	}

	h.log.Debug("sse: subscriber added", "session", sessionID, "owner", ownerSub)

	unsub := func() {
		h.mu.Lock()
		list := h.subs[sessionID]
		updated := list[:0]
		found := false
		for _, s := range list {
			if s != sub {
				updated = append(updated, s)
			} else {
				found = true
			}
		}
		if found {
			if len(updated) == 0 {
				delete(h.subs, sessionID)
			} else {
				h.subs[sessionID] = updated
			}
		}
		// Always safe to call; sync.Once ensures exactly one close.
		sub.safeClose()
		last := found && len(updated) == 0
		h.mu.Unlock()

		if found {
			metricSubscribers.Dec()
		}
		if last {
			h.cancelRedisSubscription(sessionID)
		}

		h.log.Debug("sse: subscriber removed", "session", sessionID, "owner", ownerSub)
	}

	return sub.ch, unsub, nil
}

// cancelRedisSubscription cancels the per-session Redis subscription goroutine, if any.
func (h *Hub) cancelRedisSubscription(sessionID string) {
	h.sessionsMu.Lock()
	defer h.sessionsMu.Unlock()
	if cancel, ok := h.sessions[sessionID]; ok {
		cancel()
		delete(h.sessions, sessionID)
	}
}

// runRedisSubscription subscribes to the Redis Pub/Sub channel for sessionID and
// fans out received messages to all current local subscribers. It exits when ctx
// is cancelled (i.e. when all local subscribers have disconnected or been evicted).
func (h *Hub) runRedisSubscription(ctx context.Context, sessionID string) {
	ch := h.channelName(sessionID)
	pubsub := h.redis.Subscribe(ctx, ch)
	defer pubsub.Close() //nolint:errcheck

	msgCh := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				h.log.Error("sse: failed to unmarshal redis message",
					"session", sessionID,
					"error", err,
				)
				continue
			}
			h.deliverLocal(sessionID, event)
		}
	}
}

// Publish serialises event to JSON, pushes it to the Redis channel for
// sessionID, and falls back to direct local delivery if Redis is unavailable.
func (h *Hub) Publish(sessionID string, event Event) {
	b, err := json.Marshal(event)
	if err != nil {
		h.log.Error("sse: marshal error", "session", sessionID, "event", event.Type, "error", err)
		return
	}

	if err := h.redis.Publish(context.Background(), h.channelName(sessionID), b).Err(); err != nil {
		h.log.Error("sse: redis publish failed – falling back to local delivery",
			"session", sessionID, "event", event.Type, "error", err,
		)
		h.deliverLocal(sessionID, event)
	}
}

// deliverLocal fans out event directly to all local subscribers of sessionID.
// Slow subscribers are tracked; those exceeding the drop threshold (consecutive
// drops or idle timeout) are automatically evicted.
func (h *Hub) deliverLocal(sessionID string, event Event) {
	h.mu.RLock()
	list := h.subs[sessionID]
	// Shadow-copy to avoid holding the read-lock during delivery.
	subs := make([]*subscriber, len(list))
	copy(subs, list)
	h.mu.RUnlock()

	var toEvict []*subscriber
	dropped := 0

	for _, s := range subs {
		select {
		case s.ch <- event:
			s.drops.Store(0)
			s.firstDropAt.Store(0)
		default:
			n := s.drops.Add(1)
			if n == 1 {
				s.firstDropAt.Store(time.Now().UnixNano())
			}
			dropped++

			firstDrop := time.Unix(0, s.firstDropAt.Load())
			if n >= maxConsecutiveDrops || time.Since(firstDrop) >= idleTimeout {
				toEvict = append(toEvict, s)
			}
		}
	}

	if dropped > 0 {
		metricDroppedEvents.Add(float64(dropped))
		h.log.Warn("sse: dropped events for slow subscribers",
			"session", sessionID,
			"event", event.Type,
			"dropped", dropped,
		)
	}

	if len(toEvict) > 0 {
		h.evict(sessionID, toEvict)
	}
}

// evict removes the given subscribers from sessionID, closes their channels, and
// cancels the Redis subscription goroutine if no local subscribers remain.
func (h *Hub) evict(sessionID string, targets []*subscriber) {
	targetSet := make(map[*subscriber]struct{}, len(targets))
	for _, t := range targets {
		targetSet[t] = struct{}{}
	}

	h.mu.Lock()
	list := h.subs[sessionID]
	kept := list[:0]
	var evicted []*subscriber
	for _, s := range list {
		if _, doEvict := targetSet[s]; doEvict {
			evicted = append(evicted, s)
		} else {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		delete(h.subs, sessionID)
	} else {
		h.subs[sessionID] = kept
	}
	last := len(kept) == 0
	h.mu.Unlock()

	for _, s := range evicted {
		s.safeClose()
		metricSubscribers.Dec()
		metricEvictedSubscribers.Inc()
		h.log.Warn("sse: evicted slow/idle subscriber",
			"session", sessionID,
			"owner", s.ownerSub,
		)
	}

	if last {
		h.cancelRedisSubscription(sessionID)
	}
}

// PublishPhaseChanged emits a phase_changed event for sessionID based on the
// updated session returned by the scheduler's AdvancePhase call.
func (h *Hub) PublishPhaseChanged(sess *domain.SeminarSession) {
	payload := PhaseChangedPayload{
		SessionID:      sess.ID,
		Phase:          string(sess.Phase),
		PhaseEndsAt:    sess.PhaseEndsAt,
		PhaseStartedAt: sess.PhaseStartedAt,
	}
	h.Publish(sess.ID, Event{Type: EventPhaseChanged, Data: payload})

	// If the session just completed, also emit a session_completed event.
	if sess.IsTerminal() {
		h.PublishSessionCompleted(sess.ID, string(sess.Status))
	}
}

// PublishTurnAdded emits a turn_added event for the given turn.
func (h *Hub) PublishTurnAdded(t *domain.SeminarTurn) {
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
