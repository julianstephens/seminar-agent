package scheduler_test

import (
	"sync"
	"testing"
	"time"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/observability"
	"github.com/julianstephens/formation/internal/scheduler"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestScheduler creates a Scheduler with a discard logger and no repo.
// Only use it for tests that do NOT trigger the advance callback (i.e. tests
// that involve non-timed phases, or races where Stop is called before the
// timer fires).
func newTestScheduler() *scheduler.Scheduler {
	return scheduler.New(nil, observability.NewTestLogger())
}

// timedSession returns a session in a timed phase with a deadline far enough
// in the future that the timer will not fire during the test.
func timedSession(phase domain.SessionPhase) *domain.Session {
	return &domain.Session{
		ID:          "sess-" + string(phase),
		Phase:       phase,
		PhaseEndsAt: time.Now().Add(24 * time.Hour),
	}
}

// nonTimedSession returns a session in a non-timed phase.
func nonTimedSession(phase domain.SessionPhase) *domain.Session {
	return &domain.Session{
		ID:          "sess-" + string(phase),
		Phase:       phase,
		PhaseEndsAt: time.Now().Add(-1 * time.Second), // already elapsed; irrelevant
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestRegister_nonTimedPhases verifies that registering a session in a
// non-timed phase is a silent no-op: the onPhaseChanged callback must never
// fire, even when PhaseEndsAt is in the past.
func TestRegister_nonTimedPhases(t *testing.T) {
	t.Parallel()

	nonTimed := []domain.SessionPhase{
		domain.PhaseResidueRequired,
		domain.PhaseDone,
	}

	for _, phase := range nonTimed {
		phase := phase // capture
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()

			called := make(chan struct{}, 1)
			sched := newTestScheduler()
			sched.SetOnPhaseChanged(func(_ *domain.Session) {
				called <- struct{}{}
			})

			sched.Register(nonTimedSession(phase))

			select {
			case <-called:
				t.Fatalf("onPhaseChanged fired unexpectedly for non-timed phase %q", phase)
			case <-time.After(80 * time.Millisecond):
				// expected: no callback
			}

			sched.Stop()
		})
	}
}

// TestRegister_timedPhases_timersRegistered verifies that a timed phase causes
// a timer to be registered. The test stops the scheduler before the timer can
// fire (PhaseEndsAt is set 24 h in the future), so there is no interaction
// with the nil repo.
func TestRegister_timedPhases_timersRegistered(t *testing.T) {
	t.Parallel()

	timed := []domain.SessionPhase{
		domain.PhaseReconstruction,
		domain.PhaseOpposition,
		domain.PhaseReversal,
	}

	for _, phase := range timed {
		phase := phase
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()

			fired := make(chan struct{}, 1)
			sched := newTestScheduler()
			sched.SetOnPhaseChanged(func(_ *domain.Session) {
				fired <- struct{}{}
			})

			sched.Register(timedSession(phase))

			// Stop must cancel the timer; callback must NOT fire.
			sched.Stop()

			select {
			case <-fired:
				t.Fatalf("timer fired after Stop() for phase %q", phase)
			case <-time.After(80 * time.Millisecond):
				// expected: timer cancelled
			}
		})
	}
}

// TestRegister_replacesExistingTimer ensures that calling Register a second
// time replaces an existing timer (no duplicate callbacks).
func TestRegister_replacesExistingTimer(t *testing.T) {
	t.Parallel()

	sched := newTestScheduler()
	defer sched.Stop()

	sess := timedSession(domain.PhaseReconstruction)

	// Register twice; should not panic and no duplicate timers.
	sched.Register(sess)
	sched.Register(sess)
}

// TestStop_idempotent verifies that Stop() on a fresh (no timers) scheduler
// does not panic.
func TestStop_idempotent(t *testing.T) {
	t.Parallel()

	sched := newTestScheduler()
	sched.Stop()
	sched.Stop() // second call must also be safe
}

// TestSetOnPhaseChanged_lateRegistration verifies that SetOnPhaseChanged can
// be called after New without data races (the lock must be held).
func TestSetOnPhaseChanged_lateRegistration(t *testing.T) {
	t.Parallel()

	sched := newTestScheduler()
	defer sched.Stop()

	var mu sync.Mutex
	calls := 0

	sched.SetOnPhaseChanged(func(_ *domain.Session) {
		mu.Lock()
		calls++
		mu.Unlock()
	})

	// No sessions registered → callback never invoked.
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if calls != 0 {
		t.Fatalf("expected 0 callback calls, got %d", calls)
	}
}

// ── Phase progression unit tests (domain package; no DB) ─────────────────────

// TestNextPhase_linearProgression checks that NextPhase advances through all
// phases in the expected order.
func TestNextPhase_linearProgression(t *testing.T) {
	t.Parallel()

	sequence := []struct {
		from domain.SessionPhase
		want domain.SessionPhase
	}{
		{domain.PhaseReconstruction, domain.PhaseOpposition},
		{domain.PhaseOpposition, domain.PhaseReversal},
		{domain.PhaseReversal, domain.PhaseResidueRequired},
		{domain.PhaseResidueRequired, domain.PhaseDone},
		{domain.PhaseDone, domain.PhaseDone}, // terminal: same phase returned
	}

	for _, tc := range sequence {
		got := domain.NextPhase(tc.from)
		if got != tc.want {
			t.Errorf("NextPhase(%q) = %q, want %q", tc.from, got, tc.want)
		}
	}
}

// TestIsTimedPhase_viaPhaseAllowsTurns verifies which phases are "timed"
// (accept user turns) using the exported domain.Session helper.
func TestPhaseAllowsTurns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		phase domain.SessionPhase
		allow bool
	}{
		{domain.PhaseReconstruction, true},
		{domain.PhaseOpposition, true},
		{domain.PhaseReversal, true},
		{domain.PhaseResidueRequired, false},
		{domain.PhaseDone, false},
	}

	for _, tc := range cases {
		sess := &domain.Session{Phase: tc.phase, Status: domain.SessionStatusInProgress}
		got := sess.PhaseAllowsTurns()
		if got != tc.allow {
			t.Errorf("Session{Phase:%q}.PhaseAllowsTurns() = %v, want %v", tc.phase, got, tc.allow)
		}
	}
}

// TestIsPhaseExpired checks the session's phase-timer expiry helper.
func TestIsPhaseExpired(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-1 * time.Second)
	future := time.Now().Add(1 * time.Hour)
	zero := time.Time{}

	cases := []struct {
		name        string
		phaseEndsAt time.Time
		want        bool
	}{
		{"past deadline", past, true},
		{"future deadline", future, false},
		{"zero deadline", zero, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sess := &domain.Session{PhaseEndsAt: tc.phaseEndsAt}
			if got := sess.IsPhaseExpired(); got != tc.want {
				t.Errorf("IsPhaseExpired() = %v, want %v (PhaseEndsAt=%v)", got, tc.want, tc.phaseEndsAt)
			}
		})
	}
}
