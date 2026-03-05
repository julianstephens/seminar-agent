// White-box tests for the sessions service.
// Using package service (not service_test) so we can test unexported helpers.
package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/julianstephens/formation/internal/domain"
)

// ── countSentences ────────────────────────────────────────────────────────────

func TestCountSentences(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"single period", "One sentence.", 1},
		{"multiple periods", "One. Two. Three.", 3},
		{"exclamation and question", "Really! Is that so? Yes.", 3},
		{"ellipsis counts as one", "Hmm... interesting.", 2},
		{"trailing space after period", "One.  Two.", 2},
		{"no punctuation", "No sentence ending here", 0},
		{"mixed punctuation", "First! Second? Third.", 3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := countSentences(tc.text)
			if got != tc.want {
				t.Errorf("countSentences(%q) = %d, want %d", tc.text, got, tc.want)
			}
		})
	}
}

// ── validateResidue ───────────────────────────────────────────────────────────

// goodResidue returns a text that satisfies all residue requirements.
func goodResidue() string {
	return strings.Join([]string{
		"The thesis of the author is that knowledge requires justification.",
		"However, there is an important objection we must consider carefully.",
		"The critic challenges this by pointing to counter-examples in the text.",
		"There remains a tension between the empirical and rational accounts.",
		"This unresolved difficulty makes the argument stronger in some ways.",
		"The fundamental argument persists even under scrutiny.",
	}, " ")
}

func TestValidateResidue_valid(t *testing.T) {
	t.Parallel()

	if err := validateResidue(goodResidue()); err != nil {
		t.Errorf("validateResidue(goodResidue) = %v, want nil", err)
	}
}

func TestValidateResidue_blank(t *testing.T) {
	t.Parallel()

	err := validateResidue("   ")
	if err == nil {
		t.Fatal("expected error for blank input, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}
	if ve.Field != "residue_text" {
		t.Errorf("field = %q, want %q", ve.Field, "residue_text")
	}
}

func TestValidateResidue_tooFewSentences(t *testing.T) {
	t.Parallel()

	// 4 sentences with required keywords but below the 5-sentence minimum.
	text := "The thesis is clear. However, there is an objection to consider. " +
		"A tension remains unresolved here. The argument holds."

	err := validateResidue(text)
	if err == nil {
		t.Fatal("expected error for <5 sentences, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}
	if !strings.Contains(ve.Message, "5 sentences") {
		t.Errorf("message %q should mention '5 sentences'", ve.Message)
	}
}

func TestValidateResidue_missingThesisComponent(t *testing.T) {
	t.Parallel()

	// Five sentences but no thesis/argument keyword.
	text := "There is an objection to be considered carefully here. " +
		"The counter has some merit and challenges our view. " +
		"There is a tension between the two accounts. " +
		"It remains unresolved in the literature. " +
		"This difficulty is worth exploring further."

	err := validateResidue(text)
	if err == nil {
		t.Fatal("expected error for missing thesis component, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}
	if !strings.Contains(ve.Message, "thesis") {
		t.Errorf("message %q should mention 'thesis'", ve.Message)
	}
}

func TestValidateResidue_missingObjectionComponent(t *testing.T) {
	t.Parallel()

	// Five sentences with thesis and tension but no objection keyword.
	text := "The thesis of the author argues for empiricism. " +
		"The position is clear and well-supported throughout. " +
		"There remains a tension between the two views. " +
		"The contradiction is evident in paradox found in the middle section. " +
		"This unresolved matter deserves more attention."

	err := validateResidue(text)
	if err == nil {
		t.Fatal("expected error for missing objection component, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}
	if !strings.Contains(ve.Message, "objection") {
		t.Errorf("message %q should mention 'objection'", ve.Message)
	}
}

func TestValidateResidue_missingTensionComponent(t *testing.T) {
	t.Parallel()

	// Five sentences with thesis and objection but no tension keyword.
	text := "The thesis argues that virtue is learnable. " +
		"However, there is a clear objection from critics who disagree. " +
		"The counter challenges the very foundation of this claim. " +
		"The argument is complete and coherent in its structure. " +
		"We must consider the whole carefully."

	err := validateResidue(text)
	if err == nil {
		t.Fatal("expected error for missing tension component, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}
	if !strings.Contains(ve.Message, "tension") {
		t.Errorf("message %q should mention 'tension'", ve.Message)
	}
}

// ── AssertTurnAllowed ─────────────────────────────────────────────────────────

// sessionWith returns a minimal Session for testing AssertTurnAllowed.
func sessionWith(status domain.SessionStatus, phase domain.SessionPhase, endsAt time.Time) *domain.Session {
	return &domain.Session{
		Status:      status,
		Phase:       phase,
		PhaseEndsAt: endsAt,
	}
}

var (
	future = time.Now().Add(1 * time.Hour)
	past   = time.Now().Add(-1 * time.Second)
)

func TestAssertTurnAllowed_timedPhaseNotExpired(t *testing.T) {
	t.Parallel()

	timedPhases := []domain.SessionPhase{
		domain.PhaseReconstruction,
		domain.PhaseOpposition,
		domain.PhaseReversal,
	}
	for _, phase := range timedPhases {
		phase := phase
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()
			sess := sessionWith(domain.SessionStatusInProgress, phase, future)
			if err := AssertTurnAllowed(sess); err != nil {
				t.Errorf("AssertTurnAllowed() = %v, want nil for timed in-progress phase", err)
			}
		})
	}
}

func TestAssertTurnAllowed_terminalStatus(t *testing.T) {
	t.Parallel()

	terminal := []domain.SessionStatus{
		domain.SessionStatusComplete,
		domain.SessionStatusAbandoned,
	}
	for _, status := range terminal {
		status := status
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			sess := sessionWith(status, domain.PhaseReconstruction, future)
			err := AssertTurnAllowed(sess)
			if err == nil {
				t.Fatal("expected error for terminal session, got nil")
			}
			var te *ErrSessionTerminalError
			if !errors.As(err, &te) {
				t.Errorf("error type = %T, want *ErrSessionTerminalError", err)
			}
		})
	}
}

func TestAssertTurnAllowed_nonTurnPhase(t *testing.T) {
	t.Parallel()

	nonTurnPhases := []domain.SessionPhase{
		domain.PhaseResidueRequired,
		domain.PhaseDone,
	}
	for _, phase := range nonTurnPhases {
		phase := phase
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()
			sess := sessionWith(domain.SessionStatusInProgress, phase, future)
			err := AssertTurnAllowed(sess)
			if err == nil {
				t.Fatal("expected error for non-turn phase, got nil")
			}
			var pe *ErrPhaseNoTurnsError
			if !errors.As(err, &pe) {
				t.Errorf("error type = %T, want *ErrPhaseNoTurnsError", err)
			}
		})
	}
}

func TestAssertTurnAllowed_expiredPhase(t *testing.T) {
	t.Parallel()

	sess := sessionWith(domain.SessionStatusInProgress, domain.PhaseReconstruction, past)
	err := AssertTurnAllowed(sess)
	if err == nil {
		t.Fatal("expected error for expired phase, got nil")
	}
	var ee *ErrPhaseExpiredError
	if !errors.As(err, &ee) {
		t.Errorf("error type = %T, want *ErrPhaseExpiredError", err)
	}
	if ee.Phase != domain.PhaseReconstruction {
		t.Errorf("phase = %q, want %q", ee.Phase, domain.PhaseReconstruction)
	}
}

// ── excerptHash ───────────────────────────────────────────────────────────────

func TestExcerptHash_emptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	if got := excerptHash(""); got != "" {
		t.Errorf("excerptHash(\"\") = %q, want \"\"", got)
	}
}

func TestExcerptHash_deterministicAndNonEmpty(t *testing.T) {
	t.Parallel()
	h1 := excerptHash("hello world")
	h2 := excerptHash("hello world")
	if h1 == "" {
		t.Error("excerptHash returned empty for non-empty input")
	}
	if h1 != h2 {
		t.Error("excerptHash is not deterministic")
	}
}

func TestExcerptHash_differentiatesInputs(t *testing.T) {
	t.Parallel()
	if excerptHash("a") == excerptHash("b") {
		t.Error("different inputs should produce different hashes")
	}
}
