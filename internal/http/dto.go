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

// ── Turn request DTOs ──────────────────────────────────────────────────────────

// SubmitTurnRequest is the body for POST /v1/sessions/:id/turns.
type SubmitTurnRequest struct {
	Text string `json:"text" binding:"required"`
}

// ── Turn response DTOs ─────────────────────────────────────────────────────────

// SubmitTurnResponse is returned by POST /v1/sessions/:id/turns.
// AgentTurn is nil when the agent client is not yet configured.
type SubmitTurnResponse struct {
	UserTurn  TurnResponse  `json:"user_turn"`
	AgentTurn *TurnResponse `json:"agent_turn,omitempty"`
}
