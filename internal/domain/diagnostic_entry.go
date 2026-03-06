package domain

import "time"

// DiagnosticPatternCode identifies specific reasoning patterns observed in tutorial sessions
type DiagnosticPatternCode string

const (
	PatternUndefinedTerms      DiagnosticPatternCode = "UNDEFINED_TERMS"
	PatternTextDrift           DiagnosticPatternCode = "TEXT_DRIFT"
	PatternHiddenPremises      DiagnosticPatternCode = "HIDDEN_PREMISES"
	PatternWeakStructure       DiagnosticPatternCode = "WEAK_STRUCTURE"
	PatternRhetoricalInflation DiagnosticPatternCode = "RHETORICAL_INFLATION"
	PatternPrematureSynthesis  DiagnosticPatternCode = "PREMATURE_SYNTHESIS"
)

// DiagnosticStatus tracks the lifecycle of a pattern
type DiagnosticStatus string

const (
	DiagnosticActive    DiagnosticStatus = "active"
	DiagnosticImproving DiagnosticStatus = "improving"
	DiagnosticResolved  DiagnosticStatus = "resolved"
)

// DiagnosticEvidence is a single piece of evidence for a pattern
type DiagnosticEvidence struct {
	ArtifactID    string `json:"artifact_id"`
	ArtifactTitle string `json:"artifact_title"`
	Excerpt       string `json:"excerpt"`
	Reason        string `json:"reason"`
}

// DiagnosticEntry is the atomic record of one observed reasoning pattern in one session
type DiagnosticEntry struct {
	ID                string
	TutorialID        string
	TutorialSessionID string
	OwnerSub          string
	WeekOf            time.Time
	PatternCode       DiagnosticPatternCode
	Severity          int
	Status            DiagnosticStatus
	Evidence          []DiagnosticEvidence
	Notes             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
