// Package export defines the denormalized data types and multi-format renderers
// used by the seminar and session export endpoints.
//
// Two formats are supported:
//   - JSON  (RenderSeminarJSON  / RenderSessionJSON)
//   - Markdown (RenderSeminarMarkdown / RenderSessionMarkdown)
//
// The renderers accept the domain types directly; no additional transformation
// is needed in the service or handler layers.
package export

import "github.com/julianstephens/formation/internal/domain"

// SeminarExport is the full denormalized export payload for a seminar.
// It bundles the seminar record, its complete thesis revision history, and
// every session (each with its ordered turn list).
type SeminarExport struct {
	Seminar  domain.Seminar  `json:"seminar"`
	Sessions []SessionExport `json:"sessions"`
}

// SessionExport is the full denormalized export payload for a single session.
// It bundles the session record and its ordered turn list.
type SessionExport struct {
	Session domain.Session `json:"session"`
	Turns   []domain.Turn  `json:"turns"`
}

// TutorialExport is the full denormalized export payload for a tutorial.
// It bundles the tutorial record and every session (each with its ordered turn list).
type TutorialExport struct {
	Tutorial domain.Tutorial         `json:"tutorial"`
	Sessions []TutorialSessionExport `json:"sessions"`
}

// TutorialSessionExport is the full denormalized export payload for a single tutorial session.
// It bundles the session record and its ordered turn list.
type TutorialSessionExport struct {
	Session   domain.TutorialSession `json:"session"`
	Turns     []domain.TutorialTurn  `json:"turns"`
	Artifacts []domain.Artifact      `json:"artifacts"`
}
