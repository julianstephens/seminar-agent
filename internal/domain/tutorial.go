// Package domain contains pure business-logic types with no framework dependencies.
package domain

import "time"

// ── Tutorial ───────────────────────────────────────────────────────────────────

// Tutorial represents a user-owned tutorial.
type Tutorial struct {
	ID          string    `json:"id"`
	OwnerSub    string    `json:"-"`
	Title       string    `json:"title"`
	Subject     string    `json:"subject"`
	Description string    `json:"description,omitempty"`
	Difficulty  string    `json:"difficulty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TutorialPatch carries optional fields for a partial tutorial update.
// Nil pointer fields are left unchanged in the database.
type TutorialPatch struct {
	Title       *string
	Subject     *string
	Description *string
	Difficulty  *string
}

// ── TutorialSession ────────────────────────────────────────────────────────────

// TutorialSessionStatus represents the lifecycle state of a tutorial session.
type TutorialSessionStatus string

const (
	TutorialSessionStatusInProgress TutorialSessionStatus = "in_progress"
	TutorialSessionStatusComplete   TutorialSessionStatus = "complete"
	TutorialSessionStatusAbandoned  TutorialSessionStatus = "abandoned"
)

// ValidTutorialSessionStatus reports whether s is a recognized status value.
func ValidTutorialSessionStatus(s TutorialSessionStatus) bool {
	switch s {
	case TutorialSessionStatusInProgress, TutorialSessionStatusComplete, TutorialSessionStatusAbandoned:
		return true
	}
	return false
}

// TutorialSessionKind represents the type of tutorial session.
type TutorialSessionKind string

const (
	TutorialSessionKindDiagnostic TutorialSessionKind = "diagnostic"
	TutorialSessionKindExtended   TutorialSessionKind = "extended"
)

// ValidTutorialSessionKind reports whether k is a recognized kind value.
func ValidTutorialSessionKind(k TutorialSessionKind) bool {
	switch k {
	case TutorialSessionKindDiagnostic, TutorialSessionKindExtended:
		return true
	}
	return false
}

// TutorialSession is a single tutorial session owned by a user.
type TutorialSession struct {
	ID         string                `json:"id"`
	TutorialID string                `json:"tutorial_id"`
	OwnerSub   string                `json:"-"`
	Status     TutorialSessionStatus `json:"status"`
	Kind       TutorialSessionKind   `json:"kind,omitempty"`
	Notes      string                `json:"notes,omitempty"`
	StartedAt  time.Time             `json:"started_at"`
	EndedAt    *time.Time            `json:"ended_at,omitempty"`
}

// IsTerminal reports whether the session has reached a terminal state.
func (s *TutorialSession) IsTerminal() bool {
	return s.Status == TutorialSessionStatusComplete || s.Status == TutorialSessionStatusAbandoned
}

// ── Artifact ───────────────────────────────────────────────────────────────────

// ArtifactKind describes the type of content an artifact holds.
type ArtifactKind string

const (
	ArtifactKindSummary            ArtifactKind = "summary"
	ArtifactKindNotes              ArtifactKind = "notes"
	ArtifactKindProblemSet         ArtifactKind = "problem_set"
	ArtifactKindProblemSetResponse ArtifactKind = "problem_set_response"
	ArtifactKindDiagnostic         ArtifactKind = "diagnostic"
)

// ValidArtifactKind reports whether k is a recognized kind value.
func ValidArtifactKind(k ArtifactKind) bool {
	switch k {
	case ArtifactKindSummary, ArtifactKindNotes, ArtifactKindProblemSet, ArtifactKindProblemSetResponse, ArtifactKindDiagnostic:
		return true
	}
	return false
}

// Artifact is a piece of content produced during a tutorial session.
type Artifact struct {
	ID        string       `json:"id"`
	SessionID string       `json:"session_id"`
	OwnerSub  string       `json:"-"`
	Kind      ArtifactKind `json:"kind"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	CreatedAt time.Time    `json:"created_at"`
}

// ── TutorialTurn ───────────────────────────────────────────────────────────────

// TutorialTurn is a single message within a tutorial session conversation.
type TutorialTurn struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Speaker   string    `json:"speaker"` // "user" | "agent" | "system"
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
