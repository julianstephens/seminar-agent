// Package agent provides prompt assembly utilities for the Seminar backend.
// The Assembler loads canonical.yml and rewrite.yml verbatim from the embedded
// prompt directory and composes ordered OpenAI-compatible message slices from
// structured runtime parameters.
package agent

import (
	"embed"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/julianstephens/formation/internal/domain"
)

//go:embed prompts/seminar/canonical.yml prompts/seminar/rewrite.yml prompts/tutorial/canonical.yml
var promptFS embed.FS

// ── YAML schemas ──────────────────────────────────────────────────────────────

type canonicalPrompt struct {
	Version               string            `yaml:"version"`
	Name                  string            `yaml:"name"`
	Description           string            `yaml:"description"`
	CoreSystem            string            `yaml:"core_system"`
	ModeAddenda           map[string]string `yaml:"mode_addenda"`
	PhaseAddenda          map[string]string `yaml:"phase_addenda"`
	ResidueRequirement    string            `yaml:"residue_requirement"`
	SessionHeaderTemplate string            `yaml:"session_header_template"`
}

type rewritePrompt struct {
	Version             string `yaml:"version"`
	Name                string `yaml:"name"`
	RewriteSystem       string `yaml:"rewrite_system"`
	RewriteUserTemplate string `yaml:"rewrite_user_template"`
}

type tutorialCanonicalPrompt struct {
	Version               string            `yaml:"version"`
	Name                  string            `yaml:"name"`
	Description           string            `yaml:"description"`
	CoreSystem            string            `yaml:"core_system"`
	ResponseContract      string            `yaml:"response_contract"`
	SessionKindAddenda    map[string]string `yaml:"session_kind_addenda"`
	TaskAddenda           map[string]string `yaml:"task_addenda"`
	RepairVsExerciseRule  string            `yaml:"repair_vs_exercise_rule"`
	EvidenceRules         string            `yaml:"evidence_rules"`
	PatternLexicon        string            `yaml:"pattern_lexicon"`
	PrescriptionRules     string            `yaml:"prescription_rules"`
	SessionHeaderTemplate string            `yaml:"session_header_template"`
	OptionalAddenda       map[string]string `yaml:"optional_addenda"`
}

// ── Message ───────────────────────────────────────────────────────────────────

// Message is a single prompt message compatible with the OpenAI Chat API.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ── Assembler ─────────────────────────────────────────────────────────────────

// Assembler parses and caches the canonical prompt files.
// Create a single instance at startup and share it across requests.
type Assembler struct {
	canonical *canonicalPrompt
	rewrite   *rewritePrompt
}

// NewAssembler parses canonical.yml and rewrite.yml from the embedded FS.
// Returns an error if either file cannot be parsed or is structurally invalid.
func NewAssembler() (*Assembler, error) {
	canon, err := loadCanonical()
	if err != nil {
		return nil, fmt.Errorf("load canonical prompt: %w", err)
	}
	rw, err := loadRewrite()
	if err != nil {
		return nil, fmt.Errorf("load rewrite prompt: %w", err)
	}
	return &Assembler{canonical: canon, rewrite: rw}, nil
}

func loadCanonical() (*canonicalPrompt, error) {
	data, err := promptFS.ReadFile("prompts/seminar/canonical.yml")
	if err != nil {
		return nil, err
	}
	var p canonicalPrompt
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse canonical.yml: %w", err)
	}
	return &p, nil
}

func loadRewrite() (*rewritePrompt, error) {
	data, err := promptFS.ReadFile("prompts/seminar/rewrite.yml")
	if err != nil {
		return nil, err
	}
	var p rewritePrompt
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse rewrite.yml: %w", err)
	}
	return &p, nil
}

// ── AssembleParams ────────────────────────────────────────────────────────────

// AssembleParams carries the runtime values injected into the agent prompt.
type AssembleParams struct {
	SeminarTitle  string
	SeminarThesis string
	SectionLabel  string
	Mode          string              // "paperback" or "excerpt"
	ExcerptText   string              // non-empty when Mode == "excerpt"
	Phase         domain.SessionPhase // active phase for the addendum
	Turns         []domain.Turn       // full chronological turn list
}

// Assemble composes the ordered message slice for a session agent call.
//
// Composition order (per canonical.yml notes_for_backend):
//  1. core_system  (+ mode_addendum + phase_addendum combined as single system message)
//  2. session_header (runtime-interpolated system message)
//  3. conversation turns in chronological order
//
// Returns an error when mode has no matching addendum in the YAML.
// A missing phase addendum (e.g. residue_required, done) is silently omitted
// so the pipeline does not hard-fail for terminal phases.
func (a *Assembler) Assemble(p AssembleParams) ([]Message, error) {
	modeText, ok := a.canonical.ModeAddenda[p.Mode]
	if !ok {
		return nil, fmt.Errorf("no mode_addendum for mode %q in canonical.yml", p.Mode)
	}

	phaseText := a.canonical.PhaseAddenda[string(p.Phase)] // empty string is fine

	// Combine core, mode, and phase into a single system message.
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(a.canonical.CoreSystem))
	sb.WriteString("\n\n")
	sb.WriteString(strings.TrimSpace(modeText))
	if phaseText != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(phaseText))
	}

	messages := []Message{
		{Role: "system", Content: sb.String()},
		{Role: "system", Content: a.interpolateHeader(p)},
	}

	// Append conversation history.
	for _, t := range p.Turns {
		messages = append(messages, Message{
			Role:    speakerToRole(t.Speaker),
			Content: t.Text,
		})
	}

	return messages, nil
}

// ── RewriteParams ─────────────────────────────────────────────────────────────

// RewriteParams carries the inputs for a compliance rewrite call.
type RewriteParams struct {
	OriginalOutput string
	ActivePhase    string
	ActiveMode     string
	ViolationList  []string
}

// AssembleRewrite builds the two-message slice (system + user) for a compliance
// rewrite call. The caller should pass this to the agent client and replace the
// original output with the response.
func (a *Assembler) AssembleRewrite(p RewriteParams) []Message {
	violations := strings.Join(p.ViolationList, "\n- ")
	if violations != "" {
		violations = "- " + violations
	}

	userContent := strings.NewReplacer(
		"{{original_agent_output}}", p.OriginalOutput,
		"{{active_phase}}", p.ActivePhase,
		"{{active_mode}}", p.ActiveMode,
		"{{violation_list}}", violations,
	).Replace(a.rewrite.RewriteUserTemplate)

	return []Message{
		{Role: "system", Content: strings.TrimSpace(a.rewrite.RewriteSystem)},
		{Role: "user", Content: userContent},
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (a *Assembler) interpolateHeader(p AssembleParams) string {
	return strings.TrimSpace(strings.NewReplacer(
		"{{seminar_title}}", p.SeminarTitle,
		"{{seminar_thesis}}", p.SeminarThesis,
		"{{section_label}}", p.SectionLabel,
		"{{mode}}", p.Mode,
		"{{excerpt_text}}", p.ExcerptText,
	).Replace(a.canonical.SessionHeaderTemplate))
}

func speakerToRole(speaker string) string {
	switch speaker {
	case "agent":
		return "assistant"
	case "system":
		return "system"
	default:
		return "user"
	}
}

// ── TutorialAssembler ─────────────────────────────────────────────────────────

// TutorialAssembler parses and caches the tutorial canonical prompt file.
type TutorialAssembler struct {
	canonical *tutorialCanonicalPrompt
}

// NewTutorialAssembler parses canonical.yml from the tutorial prompts directory.
func NewTutorialAssembler() (*TutorialAssembler, error) {
	canon, err := loadTutorialCanonical()
	if err != nil {
		return nil, fmt.Errorf("load tutorial canonical prompt: %w", err)
	}
	return &TutorialAssembler{canonical: canon}, nil
}

func loadTutorialCanonical() (*tutorialCanonicalPrompt, error) {
	data, err := promptFS.ReadFile("prompts/tutorial/canonical.yml")
	if err != nil {
		return nil, err
	}
	var p tutorialCanonicalPrompt
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse tutorial canonical.yml: %w", err)
	}
	return &p, nil
}

// ── TutorialAssembleParams ────────────────────────────────────────────────────

// TutorialAssembleParams carries the runtime values for tutorial prompts.
type TutorialAssembleParams struct {
	TutorialTitle      string
	SessionKind        string // e.g. "diagnostic", "extended"
	TaskMode           string // e.g. "review_only", "problemset_generation"
	WeekOf             string
	Artifacts          string
	PriorDiagnostics   string
	PreviousProblemSet string
	ProblemSetResponse string
	Turns              []domain.TutorialTurn
}

// AssembleTutorial composes the ordered message slice for a tutorial session.
//
// Composition order (per canonical.yml v1.1.0):
//  1. core_system
//  2. response_contract
//  3. session_kind_addendum
//  4. task_addendum
//  5. evidence_rules
//  6. prescription_rules
//  7. session_header (runtime-interpolated)
//  8. conversation turns
//  9. optional_addenda (only for cold starts)
//
// The example_only addendum is appended after conversation turns only for cold starts.
func (a *TutorialAssembler) AssembleTutorial(p TutorialAssembleParams) ([]Message, error) {
	// Get session kind addendum (optional - defaults to empty).
	sessionKindText := a.canonical.SessionKindAddenda[p.SessionKind]

	// Get task addendum (optional - defaults to empty).
	taskText := a.canonical.TaskAddenda[p.TaskMode]

	// Compose core system message per canonical order.
	var sb strings.Builder

	// 1. core_system
	sb.WriteString(strings.TrimSpace(a.canonical.CoreSystem))

	// 2. response_contract
	if a.canonical.ResponseContract != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(a.canonical.ResponseContract))
	}

	// 3. session_kind_addendum
	if sessionKindText != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(sessionKindText))
	}

	// 4. task_addendum
	if taskText != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(taskText))
	}

	// 5. evidence_rules
	if a.canonical.EvidenceRules != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(a.canonical.EvidenceRules))
	}

	// 6. prescription_rules
	if a.canonical.PrescriptionRules != "" {
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(a.canonical.PrescriptionRules))
	}

	// 7. session_header (as separate system message)
	messages := []Message{
		{Role: "system", Content: sb.String()},
		{Role: "system", Content: a.interpolateTutorialHeader(p)},
	}

	// 8. conversation turns
	for _, t := range p.Turns {
		messages = append(messages, Message{
			Role:    speakerToRole(t.Speaker),
			Content: t.Text,
		})
	}

	// 9. optional_addenda (example_only for cold starts only)
	// Appended after conversation turns to avoid biasing ongoing sessions.
	if len(p.Turns) == 0 {
		if exampleText, ok := a.canonical.OptionalAddenda["example_only"]; ok && exampleText != "" {
			messages = append(messages, Message{
				Role:    "system",
				Content: strings.TrimSpace(exampleText),
			})
		}
	}

	return messages, nil
}

// ── tutorial helpers ──────────────────────────────────────────────────────────

func (a *TutorialAssembler) interpolateTutorialHeader(p TutorialAssembleParams) string {
	return strings.TrimSpace(strings.NewReplacer(
		"{{tutorial_title}}", p.TutorialTitle,
		"{{session_kind}}", p.SessionKind,
		"{{task_mode}}", p.TaskMode,
		"{{week_of}}", p.WeekOf,
		"{{artifacts}}", p.Artifacts,
		"{{prior_diagnostics}}", p.PriorDiagnostics,
		"{{previous_problem_set}}", p.PreviousProblemSet,
		"{{problem_set_response}}", p.ProblemSetResponse,
	).Replace(a.canonical.SessionHeaderTemplate))
}
