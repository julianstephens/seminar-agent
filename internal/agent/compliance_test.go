package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/julianstephens/formation/internal/agent"
)

// ── stub provider ─────────────────────────────────────────────────────────────

// stubProvider implements agent.Provider for use in tests. It returns a fixed
// completion string or a fixed error on every call.
type stubProvider struct {
	reply string
	err   error
}

// CompleteStream implements agent.Provider.
func (s *stubProvider) CompleteStream(
	ctx context.Context,
	messages []agent.Message,
	chunkFn func(chunk string) error,
) (string, error) {
	panic("unimplemented")
}

func (s *stubProvider) Complete(_ context.Context, _ []agent.Message) (string, error) {
	return s.reply, s.err
}

// ── stub assembler ────────────────────────────────────────────────────────────

// newAssembler returns a real Assembler by parsing the embedded prompt files.
// If the embedded YAML cannot be parsed the test is skipped, because this is
// an environment dependency (the embed.go includes the prompt files at build
// time).
func newAssembler(t *testing.T) *agent.Assembler {
	t.Helper()
	a, err := agent.NewAssembler()
	if err != nil {
		t.Skipf("skipping: cannot init Assembler (%v)", err)
	}
	return a
}

// ── CheckViolations ───────────────────────────────────────────────────────────

func TestCheckViolations_cleanOutput(t *testing.T) {
	t.Parallel()

	// A normal Socratic question during reconstruction should trigger no rules.
	output := "What do you think the author means when they write about knowledge? " +
		"Can you reconstruct the argument in your own words?"
	violations := agent.CheckViolations(output, "reconstruction", "paperback")
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for clean output, got %v", violations)
	}
}

func TestCheckViolations_noQuestion(t *testing.T) {
	t.Parallel()

	// During timed dialogue phases the agent must always ask a question.
	output := "That is a reasonable interpretation of the text."
	violations := agent.CheckViolations(output, "reconstruction", "paperback")
	found := containsAny(violations, "no question")
	if !found {
		t.Errorf("expected 'no_question' violation, got %v", violations)
	}
}

func TestCheckViolations_noQuestion_skippedForResidue(t *testing.T) {
	t.Parallel()

	// The no_question rule only applies during timed dialogue phases.
	output := "Good summary of the session."
	violations := agent.CheckViolations(output, "residue_required", "paperback")
	found := containsAny(violations, "no question")
	if found {
		t.Errorf("no_question rule should not fire during residue_required, got %v", violations)
	}
}

func TestCheckViolations_revealsThesis(t *testing.T) {
	t.Parallel()

	forbiddenPhrases := []string{
		"The thesis is that rationalism is correct.",
		"The correct answer is empiricism.",
		"The answer is that knowledge is justified true belief.",
		"The author argues that virtue is teachable.",
		"The book claims this to be the case.",
		"The author's main point is that…",
	}

	for _, phrase := range forbiddenPhrases {
		phrase := phrase
		t.Run(phrase[:min(30, len(phrase))], func(t *testing.T) {
			t.Parallel()
			violations := agent.CheckViolations(phrase, "reconstruction", "paperback")
			if !containsAny(violations, "thesis") {
				t.Errorf("expected reveals_thesis violation for %q, got %v", phrase, violations)
			}
		})
	}
}

func TestCheckViolations_breaksCharacter(t *testing.T) {
	t.Parallel()

	cases := []string{
		"I am an AI assistant.",
		"I'm an AI language model.",
		"I am a chatbot.",
		"I'm an LLM.",
		"I am a language model.",
	}

	for _, c := range cases {
		c := c
		t.Run(c, func(t *testing.T) {
			t.Parallel()
			violations := agent.CheckViolations(c+"?", "reconstruction", "paperback")
			if !containsAny(violations, "AI") && !containsAny(violations, "character") {
				t.Errorf("expected breaks_character violation for %q, got %v", c, violations)
			}
		})
	}
}

func TestCheckViolations_prematurePhaseMention(t *testing.T) {
	t.Parallel()

	cases := []struct {
		output string
		phase  string
		expect bool
	}{
		{"We will discuss the opposition later.", "reconstruction", true},
		{"Now let's enter the reversal phase.", "reconstruction", true},
		{"We will discuss the opposition later.", "opposition", false}, // only fires during recon
		{"Normal question about the argument?", "reconstruction", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.phase+"_"+tc.output[:min(20, len(tc.output))], func(t *testing.T) {
			t.Parallel()
			violations := agent.CheckViolations(tc.output, tc.phase, "paperback")
			// The premature_phase_mention description contains "later phase".
			found := containsAny(violations, "later phase") || containsAny(violations, "premature")
			if found != tc.expect {
				t.Errorf("premature_phase_mention for phase=%q output=%q: got=%v want=%v violations=%v",
					tc.phase, tc.output, found, tc.expect, violations)
			}
		})
	}
}

func TestCheckViolations_emptyResponse(t *testing.T) {
	t.Parallel()

	for _, empty := range []string{"", "   ", "\t\n"} {
		empty := empty
		violations := agent.CheckViolations(empty, "reconstruction", "paperback")
		if !containsAny(violations, "blank") && !containsAny(violations, "empty") {
			t.Errorf("expected empty_response violation for %q, got %v", empty, violations)
		}
	}
}

// ── ApplyCompliance ───────────────────────────────────────────────────────────

func TestApplyCompliance_cleanOutput_noRewrite(t *testing.T) {
	t.Parallel()

	assembler := newAssembler(t)
	provider := &stubProvider{reply: "should not be called"}

	clean := "What exactly do you mean by that argument? Can you explain further?"
	result, err := agent.ApplyCompliance(
		context.Background(),
		provider,
		assembler,
		clean,
		"reconstruction",
		"paperback",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rewritten {
		t.Error("Rewritten should be false for clean output")
	}
	if result.Text != clean {
		t.Errorf("Text = %q, want unchanged %q", result.Text, clean)
	}
	if len(result.Flags) != 0 {
		t.Errorf("Flags = %v, want empty", result.Flags)
	}
}

func TestApplyCompliance_violation_rewriteSucceeds(t *testing.T) {
	t.Parallel()

	assembler := newAssembler(t)
	rewrittenText := "Can you elaborate on what you mean by that?"
	provider := &stubProvider{reply: rewrittenText}

	// This output has no question mark → triggers no_question violation.
	violating := "That is an interesting thought."
	result, err := agent.ApplyCompliance(
		context.Background(),
		provider,
		assembler,
		violating,
		"reconstruction",
		"paperback",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Rewritten {
		t.Error("Rewritten should be true when violations were fixed")
	}
	if result.Text != rewrittenText {
		t.Errorf("Text = %q, want rewritten %q", result.Text, rewrittenText)
	}
	found := false
	for _, f := range result.Flags {
		if f == agent.FlagAgentRewrite {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected flag %q in %v", agent.FlagAgentRewrite, result.Flags)
	}
}

func TestApplyCompliance_violation_providerError_returnsOriginal(t *testing.T) {
	t.Parallel()

	assembler := newAssembler(t)
	provErr := errors.New("openai unavailable")
	provider := &stubProvider{err: provErr}

	// Violating output (no question).
	violating := "That is an interesting thought."
	result, err := agent.ApplyCompliance(
		context.Background(),
		provider,
		assembler,
		violating,
		"reconstruction",
		"paperback",
	)
	if err == nil {
		t.Fatal("expected error from provider, got nil")
	}
	// Even on error, original text must be returned so the turn can persist.
	if result.Text != violating {
		t.Errorf("Text = %q, want original %q on provider error", result.Text, violating)
	}
	if result.Rewritten {
		t.Error("Rewritten should be false when provider errored")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// containsAny reports whether any violation description contains substr.
func containsAny(violations []string, substr string) bool {
	for _, v := range violations {
		if len(v) > 0 {
			// case-insensitive substring search
			lower := toLower(v)
			cmp := toLower(substr)
			if contains(lower, cmp) {
				return true
			}
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
