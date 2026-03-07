package http

import (
	"time"

	"github.com/julianstephens/formation/internal/domain"
)

// ── Seminar request DTOs ───────────────────────────────────────────────────────

// CreateSeminarRequest is the body for POST /v1/seminars.
type CreateSeminarRequest struct {
	Title               string `json:"title"                 binding:"required"`
	Author              string `json:"author"`
	EditionNotes        string `json:"edition_notes"`
	ThesisCurrent       string `json:"thesis_current"        binding:"required"`
	DefaultMode         string `json:"default_mode"`
	DefaultReconMinutes int    `json:"default_recon_minutes"`
}

// UpdateSeminarRequest is the body for PATCH /v1/seminars/:id.
// All fields are optional; nil means "leave unchanged".
type UpdateSeminarRequest struct {
	Title               *string `json:"title"`
	Author              *string `json:"author"`
	EditionNotes        *string `json:"edition_notes"`
	DefaultMode         *string `json:"default_mode"`
	DefaultReconMinutes *int    `json:"default_recon_minutes"`
}

// ── Seminar response DTOs ──────────────────────────────────────────────────────

// SeminarResponse is the JSON representation of a Seminar resource.
type SeminarResponse struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Author              string    `json:"author,omitempty"`
	EditionNotes        string    `json:"edition_notes,omitempty"`
	ThesisCurrent       string    `json:"thesis_current"`
	DefaultMode         string    `json:"default_mode"`
	DefaultReconMinutes int       `json:"default_recon_minutes"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ── Session request DTOs ───────────────────────────────────────────────────────

// CreateSessionRequest is the body for POST /v1/seminars/:id/sessions.
type CreateSessionRequest struct {
	SectionLabel string `json:"section_label" binding:"required"`
	Mode         string `json:"mode"`          // optional; falls back to seminar default
	ExcerptText  string `json:"excerpt_text"`  // required when mode == "excerpt"
	ReconMinutes int    `json:"recon_minutes"` // optional; falls back to seminar default
}

// SubmitResidueRequest is the body for POST /v1/sessions/:id/residue.
type SubmitResidueRequest struct {
	ResidueText string `json:"residue_text" binding:"required"`
}

// ── Session response DTOs ──────────────────────────────────────────────────────

// TurnResponse is the JSON representation of a single turn.
type TurnResponse struct {
	ID        string              `json:"id"`
	SessionID string              `json:"session_id"`
	Phase     domain.SessionPhase `json:"phase"`
	Speaker   string              `json:"speaker"`
	Text      string              `json:"text"`
	Flags     []string            `json:"flags"`
	CreatedAt time.Time           `json:"created_at"`
}

// SessionResponse is the JSON representation of a Session resource.
type SessionResponse struct {
	ID             string               `json:"id"`
	SeminarID      string               `json:"seminar_id"`
	SectionLabel   string               `json:"section_label"`
	Mode           string               `json:"mode"`
	ExcerptText    string               `json:"excerpt_text,omitempty"`
	ExcerptHash    string               `json:"excerpt_hash,omitempty"`
	Status         domain.SessionStatus `json:"status"`
	Phase          domain.SessionPhase  `json:"phase"`
	ReconMinutes   int                  `json:"recon_minutes"`
	PhaseStartedAt time.Time            `json:"phase_started_at"`
	PhaseEndsAt    time.Time            `json:"phase_ends_at"`
	StartedAt      time.Time            `json:"started_at"`
	EndedAt        *time.Time           `json:"ended_at,omitempty"`
	ResidueText    string               `json:"residue_text,omitempty"`
}

// SessionDetailResponse is the JSON representation of a session with its turns.
type SessionDetailResponse struct {
	SessionResponse
	Turns []TurnResponse `json:"turns"`
}

// ── Tutorial request DTOs ──────────────────────────────────────────────────────

// CreateTutorialRequest is the body for POST /v1/tutorials.
type CreateTutorialRequest struct {
	Title       string `json:"title"       binding:"required"`
	Subject     string `json:"subject"     binding:"required"`
	Description string `json:"description"`
	Difficulty  string `json:"difficulty"`
}

// UpdateTutorialRequest is the body for PATCH /v1/tutorials/:id.
// All fields are optional; nil means "leave unchanged".
type UpdateTutorialRequest struct {
	Title       *string `json:"title"`
	Subject     *string `json:"subject"`
	Description *string `json:"description"`
	Difficulty  *string `json:"difficulty"`
}

// CreateTutorialSessionRequest is the body for POST /v1/tutorials/:id/sessions.
type CreateTutorialSessionRequest struct {
	Kind string `json:"kind"`
}

// CompleteTutorialSessionRequest is the body for POST /v1/tutorial-sessions/:id/complete.
type CompleteTutorialSessionRequest struct {
	Notes string `json:"notes"`
}

// CreateArtifactRequest is the body for POST /v1/tutorial-sessions/:id/artifacts.
type CreateArtifactRequest struct {
	Kind         string `json:"kind"           binding:"required"`
	Title        string `json:"title"          binding:"required"`
	Content      string `json:"content"        binding:"required"`
	ProblemSetID string `json:"problem_set_id"`
}

// ── Tutorial response DTOs ─────────────────────────────────────────────────────

// TutorialResponse is the JSON representation of a Tutorial resource.
type TutorialResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Subject     string    `json:"subject"`
	Description string    `json:"description,omitempty"`
	Difficulty  string    `json:"difficulty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TutorialSessionResponse is the JSON representation of a TutorialSession resource.
type TutorialSessionResponse struct {
	ID         string                       `json:"id"`
	TutorialID string                       `json:"tutorial_id"`
	Status     domain.TutorialSessionStatus `json:"status"`
	Kind       domain.TutorialSessionKind   `json:"kind,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	StartedAt  time.Time                    `json:"started_at"`
	EndedAt    *time.Time                   `json:"ended_at,omitempty"`
}

// ArtifactResponse is the JSON representation of an Artifact resource.
type ArtifactResponse struct {
	ID           string              `json:"id"`
	SessionID    string              `json:"session_id"`
	Kind         domain.ArtifactKind `json:"kind"`
	Title        string              `json:"title"`
	Content      string              `json:"content"`
	ProblemSetID string              `json:"problem_set_id,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
}

// TutorialSessionDetailResponse is the JSON representation of a session with its artifacts and turns.
type TutorialSessionDetailResponse struct {
	TutorialSessionResponse
	Artifacts  []ArtifactResponse     `json:"artifacts"`
	Turns      []TutorialTurnResponse `json:"turns"`
	ProblemSet *ProblemSetResponse    `json:"problem_set,omitempty"`
}

// ── Turn request DTOs ──────────────────────────────────────────────────────────

// SubmitTurnRequest is the body for POST /v1/sessions/:id/turns.
type SubmitTurnRequest struct {
	Text string `json:"text" binding:"required"`
}

// SubmitTutorialTurnRequest is the body for POST /v1/tutorial-sessions/:id/turns.
type SubmitTutorialTurnRequest struct {
	Text string `json:"text" binding:"required"`
}

// ── Turn response DTOs ─────────────────────────────────────────────────────────

// SubmitTurnResponse is returned by POST /v1/sessions/:id/turns.
// AgentTurn is nil when the agent client is not yet configured.
type SubmitTurnResponse struct {
	UserTurn  TurnResponse  `json:"user_turn"`
	AgentTurn *TurnResponse `json:"agent_turn,omitempty"`
}

// TutorialTurnResponse is the JSON representation of a TutorialTurn resource.
type TutorialTurnResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Speaker   string    `json:"speaker"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// SubmitTutorialTurnResponse is returned by POST /v1/tutorial-sessions/:id/turns.
type SubmitTutorialTurnResponse struct {
	UserTurn  TutorialTurnResponse  `json:"user_turn"`
	AgentTurn *TutorialTurnResponse `json:"agent_turn,omitempty"`
}

// ── Diagnostic response DTOs ───────────────────────────────────────────────────

// DiagnosticEntryResponse is the JSON representation of a diagnostic entry.
type DiagnosticEntryResponse struct {
	ID                string                      `json:"id"`
	TutorialID        string                      `json:"tutorial_id"`
	TutorialSessionID string                      `json:"tutorial_session_id"`
	WeekOf            time.Time                   `json:"week_of"`
	PatternCode       string                      `json:"pattern_code"`
	Severity          int                         `json:"severity"`
	Status            string                      `json:"status"`
	Evidence          []domain.DiagnosticEvidence `json:"evidence"`
	Notes             string                      `json:"notes,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

// PatternSummaryItemResponse is the JSON representation of a pattern summary item.
type PatternSummaryItemResponse struct {
	PatternCode  string `json:"pattern_code"`
	Occurrences  int    `json:"occurrences"`
	LastSeenWeek string `json:"last_seen_week"`
	Trend        string `json:"trend"`
}

// ProblemSetTaskResponse is the JSON representation of a problem set task.
type ProblemSetTaskResponse struct {
	PatternCode string `json:"pattern_code"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// ProblemSetResponse is the JSON representation of a problem set.
type ProblemSetResponse struct {
	ID                    string                   `json:"id"`
	TutorialID            string                   `json:"tutorial_id"`
	WeekOf                time.Time                `json:"week_of"`
	AssignedFromSessionID string                   `json:"assigned_from_session_id,omitempty"`
	Status                string                   `json:"status"`
	Tasks                 []ProblemSetTaskResponse `json:"tasks"`
	ReviewNotes           string                   `json:"review_notes,omitempty"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

// ListDiagnosticsResponse is returned by GET /v1/tutorials/:id/diagnostics.
type ListDiagnosticsResponse struct {
	Entries []DiagnosticEntryResponse `json:"entries"`
}

// DiagnosticSummaryResponse is returned by GET /v1/tutorials/:id/diagnostics/summary.
type DiagnosticSummaryResponse struct {
	Items []PatternSummaryItemResponse `json:"items"`
}

// ListProblemSetsResponse is returned by GET /v1/tutorials/:id/problem-sets.
type ListProblemSetsResponse struct {
	ProblemSets []ProblemSetResponse `json:"problem_sets"`
}
