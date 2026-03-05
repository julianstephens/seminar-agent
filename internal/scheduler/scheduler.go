// Package scheduler provides the authoritative single-instance phase scheduler
// for seminar sessions.
//
// Design notes
// ────────────
// A single in-process timer per session drives phase transitions. Each timer
// fires at phase_ends_at and calls AdvancePhase on the repository, which uses
// SELECT … FOR UPDATE to serialize any concurrent callers. The transition is a
// race-safe no-op when the phase has already been advanced by another path
// (e.g., the session was abandoned between the timer firing and the DB write).
//
// On restart the scheduler queries for all in-progress timed sessions and
// re-registers their timers. Sessions whose phase_ends_at has already elapsed
// are fired immediately (delay=0) so the transition happens within milliseconds
// of the server coming back online.
//
// Extension note: When horizontal scaling is needed this component can be
// extended to gate transitions through a Postgres advisory lock, keeping the
// Register/Recover interface stable.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
	"github.com/julianstephens/formation/internal/service"
)

// Scheduler manages per-session phase-advance timers.
// All exported methods are safe for concurrent use.
type Scheduler struct {
	mu     sync.Mutex
	timers map[string]*time.Timer // keyed by session ID

	sessRepo *repo.SessionRepo
	log      *slog.Logger

	// onPhaseChanged is invoked synchronously in the timer goroutine after each
	// successful phase transition. Registered by the SSE hub in a later phase.
	// Must be non-blocking.
	onPhaseChanged func(sess *domain.Session)
}

// New creates a Scheduler backed by the given SessionRepo.
func New(r *repo.SessionRepo, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		timers:         make(map[string]*time.Timer),
		sessRepo:       r,
		log:            logger,
		onPhaseChanged: func(_ *domain.Session) {}, // no-op until SSE hub (step 7)
	}
}

// SetOnPhaseChanged registers a callback that is invoked after each
// authoritative phase transition. The callback receives the updated session
// (already persisted) and must not block.
//
// This hook is the integration point for the SSE hub (step 7).
func (s *Scheduler) SetOnPhaseChanged(fn func(sess *domain.Session)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPhaseChanged = fn
}

// Register schedules a phase-advance timer for sess. If the session is already
// in a non-timed phase (residue_required, done) or is terminal, the call is a
// no-op. If a timer is already registered for the session it is replaced so
// this method is safe to call multiple times.
func (s *Scheduler) Register(sess *domain.Session) {
	if !isTimedPhase(sess.Phase) {
		return
	}

	delay := time.Until(sess.PhaseEndsAt)
	if delay < 0 {
		delay = 0
	}

	sessionID := sess.ID
	fromPhase := sess.Phase

	s.mu.Lock()
	defer s.mu.Unlock()

	if old, ok := s.timers[sessionID]; ok {
		old.Stop()
	}
	s.timers[sessionID] = time.AfterFunc(delay, func() {
		s.advance(sessionID, fromPhase)
	})

	s.log.Debug("scheduler registered timer",
		"session", sessionID,
		"phase", fromPhase,
		"delay", delay.Round(time.Second).String(),
	)
}

// RecoverInProgress queries the database for all in-progress timed sessions
// and calls Register for each one. Sessions whose phase_ends_at has already
// elapsed will fire immediately (delay=0), triggering the transition within
// milliseconds.
//
// Call this once during application startup after the database is ready.
func (s *Scheduler) RecoverInProgress(ctx context.Context) error {
	sessions, err := s.sessRepo.ListInProgress(ctx)
	if err != nil {
		return fmt.Errorf("scheduler recover in-progress sessions: %w", err)
	}

	for i := range sessions {
		s.Register(&sessions[i])
	}

	s.log.Info("scheduler recovered in-progress sessions", "count", len(sessions))
	return nil
}

// Stop cancels all pending timers. It does not wait for any in-flight timer
// callbacks to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, t := range s.timers {
		t.Stop()
		delete(s.timers, id)
	}
	s.log.Info("scheduler stopped, all timers cancelled")
}

// ── internal ──────────────────────────────────────────────────────────────────

// advance is the timer callback. It advances the phase in the database,
// inserts a system turn for the transcript, fires the onPhaseChanged hook, and
// chains a new timer for the next timed phase (if any).
func (s *Scheduler) advance(sessionID string, fromPhase domain.SessionPhase) {
	// Remove our own timer entry before doing any async work so that a
	// concurrent Register call during this callback is not clobbered.
	s.mu.Lock()
	delete(s.timers, sessionID)
	s.mu.Unlock()

	ctx := context.Background()

	sess, advanced, err := s.sessRepo.AdvancePhase(ctx, sessionID, fromPhase)
	if err != nil {
		s.log.Error("scheduler: advance phase failed",
			"session", sessionID,
			"from_phase", fromPhase,
			"err", err,
		)
		return
	}

	if !advanced {
		// Race-safe no-op: session was already advanced or abandoned.
		s.log.Debug("scheduler: advance skipped (already transitioned)",
			"session", sessionID,
			"from_phase", fromPhase,
		)
		return
	}

	s.log.Info("scheduler: phase advanced",
		"session", sessionID,
		"from", fromPhase,
		"to", sess.Phase,
	)

	// Insert a system turn so the turn transcript reflects the phase change.
	// This is non-fatal: a logging failure should not undo the transition.
	if _, err := service.InsertPhaseChangeTurn(ctx, s.sessRepo, sessionID, sess.Phase); err != nil {
		s.log.Error("scheduler: insert phase-change system turn failed",
			"session", sessionID,
			"phase", sess.Phase,
			"err", err,
		)
	}

	// Notify the SSE hub (or any other observer). Non-blocking by contract.
	s.mu.Lock()
	cb := s.onPhaseChanged
	s.mu.Unlock()
	cb(sess)

	// Chain: schedule the next timer if the new phase is also timed.
	if isTimedPhase(sess.Phase) {
		s.Register(sess)
	}
}

// isTimedPhase reports whether phase p has an automatic countdown timer.
// Only the three dialogue phases advance automatically; residue_required waits
// for a user submission and done is terminal.
func isTimedPhase(p domain.SessionPhase) bool {
	switch p {
	case domain.PhaseReconstruction, domain.PhaseOpposition, domain.PhaseReversal:
		return true
	}
	return false
}
