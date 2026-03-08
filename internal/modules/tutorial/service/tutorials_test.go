package service

import (
	"testing"
	"time"

	"github.com/julianstephens/formation/internal/domain"
)

// ── sundayOfWeek ──────────────────────────────────────────────────────────────

func TestSundayOfWeek_sundayInput(t *testing.T) {
	t.Parallel()
	// 2024-03-10 is a Sunday.
	sunday := time.Date(2024, 3, 10, 14, 0, 0, 0, time.UTC)
	got := sundayOfWeek(sunday)
	want := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("sundayOfWeek(%v) = %v, want %v", sunday, got, want)
	}
}

func TestSundayOfWeek_midweek(t *testing.T) {
	t.Parallel()
	// 2024-03-13 is a Wednesday; its week's Sunday is 2024-03-10.
	wednesday := time.Date(2024, 3, 13, 9, 30, 0, 0, time.UTC)
	got := sundayOfWeek(wednesday)
	want := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("sundayOfWeek(%v) = %v, want %v", wednesday, got, want)
	}
}

func TestSundayOfWeek_saturday(t *testing.T) {
	t.Parallel()
	// 2024-03-16 is a Saturday; its week's Sunday is 2024-03-10.
	saturday := time.Date(2024, 3, 16, 23, 59, 59, 0, time.UTC)
	got := sundayOfWeek(saturday)
	want := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("sundayOfWeek(%v) = %v, want %v", saturday, got, want)
	}
}

func TestSundayOfWeek_monday(t *testing.T) {
	t.Parallel()
	// 2024-03-11 is a Monday; its week's Sunday is 2024-03-10.
	monday := time.Date(2024, 3, 11, 8, 0, 0, 0, time.UTC)
	got := sundayOfWeek(monday)
	want := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("sundayOfWeek(%v) = %v, want %v", monday, got, want)
	}
}

// ── parseAndValidateTutorialCommand ──────────────────────────────────────────

func makeSession(kind domain.TutorialSessionKind, status domain.TutorialSessionStatus) *domain.TutorialSession {
	return &domain.TutorialSession{
		ID:     "sess-1",
		Status: status,
		Kind:   kind,
	}
}

func TestParseAndValidate_noCommand(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindDiagnostic, domain.TutorialSessionStatusInProgress)
	cmd, err := parseAndValidateTutorialCommand("Hello, here are my notes.", sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != tutorialCommandNone {
		t.Errorf("expected commandNone, got %v", cmd)
	}
}

func TestParseAndValidate_problemSetInExtended(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindExtended, domain.TutorialSessionStatusInProgress)
	cmd, err := parseAndValidateTutorialCommand("/problem-set", sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != tutorialCommandProblemSet {
		t.Errorf("expected commandProblemSet, got %v", cmd)
	}
}

func TestParseAndValidate_problemSetInDiagnostic_rejected(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindDiagnostic, domain.TutorialSessionStatusInProgress)
	_, err := parseAndValidateTutorialCommand("/problem-set", sess)
	if err == nil {
		t.Fatal("expected validation error for /problem-set in diagnostic session, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestParseAndValidate_unknownCommand_rejected(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindExtended, domain.TutorialSessionStatusInProgress)
	_, err := parseAndValidateTutorialCommand("/unknown-cmd", sess)
	if err == nil {
		t.Fatal("expected validation error for unknown command, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestParseAndValidate_problemSet_withLeadingWhitespace(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindExtended, domain.TutorialSessionStatusInProgress)
	// Leading whitespace should still parse as a command.
	cmd, err := parseAndValidateTutorialCommand("  /problem-set  ", sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != tutorialCommandProblemSet {
		t.Errorf("expected commandProblemSet, got %v", cmd)
	}
}

func TestParseAndValidate_bareSlash_rejected(t *testing.T) {
	t.Parallel()
	sess := makeSession(domain.TutorialSessionKindExtended, domain.TutorialSessionStatusInProgress)
	_, err := parseAndValidateTutorialCommand("/", sess)
	if err == nil {
		t.Fatal("expected validation error for bare '/', got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isValidationError(err error, target **ValidationError) bool {
	if err == nil {
		return false
	}
	v, ok := err.(*ValidationError)
	if ok && target != nil {
		*target = v
	}
	return ok
}

// ── parseProblemSetCommandOptions ────────────────────────────────────────────

func TestParseProblemSetCommandOptions_defaults(t *testing.T) {
	t.Parallel()
	opts, err := parseProblemSetCommandOptions("/problem-set")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Patterns != "auto" {
		t.Errorf("expected patterns=auto, got %s", opts.Patterns)
	}
	if opts.Difficulty != "intermediate" {
		t.Errorf("expected difficulty=intermediate, got %s", opts.Difficulty)
	}
	if opts.Mode != "commit" {
		t.Errorf("expected mode=commit, got %s", opts.Mode)
	}
}

func TestParseProblemSetCommandOptions_allOptions(t *testing.T) {
	t.Parallel()
	opts, err := parseProblemSetCommandOptions("/problem-set /patterns TEXT_DRIFT /difficulty advanced /mode preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Patterns != "TEXT_DRIFT" {
		t.Errorf("expected patterns=TEXT_DRIFT, got %s", opts.Patterns)
	}
	if opts.Difficulty != "advanced" {
		t.Errorf("expected difficulty=advanced, got %s", opts.Difficulty)
	}
	if opts.Mode != "preview" {
		t.Errorf("expected mode=preview, got %s", opts.Mode)
	}
}

func TestParseProblemSetCommandOptions_partialOptions(t *testing.T) {
	t.Parallel()
	opts, err := parseProblemSetCommandOptions("/problem-set /difficulty beginner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Patterns != "auto" {
		t.Errorf("expected patterns=auto (default), got %s", opts.Patterns)
	}
	if opts.Difficulty != "beginner" {
		t.Errorf("expected difficulty=beginner, got %s", opts.Difficulty)
	}
	if opts.Mode != "commit" {
		t.Errorf("expected mode=commit (default), got %s", opts.Mode)
	}
}

func TestParseProblemSetCommandOptions_invalidDifficulty(t *testing.T) {
	t.Parallel()
	_, err := parseProblemSetCommandOptions("/problem-set /difficulty hard")
	if err == nil {
		t.Fatal("expected validation error for invalid difficulty, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestParseProblemSetCommandOptions_invalidMode(t *testing.T) {
	t.Parallel()
	_, err := parseProblemSetCommandOptions("/problem-set /mode draft")
	if err == nil {
		t.Fatal("expected validation error for invalid mode, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestParseProblemSetCommandOptions_missingValue(t *testing.T) {
	t.Parallel()
	_, err := parseProblemSetCommandOptions("/problem-set /patterns")
	if err == nil {
		t.Fatal("expected validation error for missing patterns value, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestParseProblemSetCommandOptions_unknownOption(t *testing.T) {
	t.Parallel()
	_, err := parseProblemSetCommandOptions("/problem-set /unknown value")
	if err == nil {
		t.Fatal("expected validation error for unknown option, got nil")
	}
	var ve *ValidationError
	if !isValidationError(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

// ── mapCommandDifficultyToPromptDifficulty ───────────────────────────────────

func TestMapCommandDifficultyToPromptDifficulty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"beginner", "basic"},
		{"intermediate", "standard"},
		{"advanced", "rigorous"},
		{"invalid", "standard"}, // fallback
	}

	for _, tt := range tests {
		got := mapCommandDifficultyToPromptDifficulty(tt.input)
		if got != tt.want {
			t.Errorf("mapCommandDifficultyToPromptDifficulty(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
