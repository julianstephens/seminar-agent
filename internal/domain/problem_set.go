package domain

import "time"

// ProblemSetTask represents a single exercise in a problem set
type ProblemSetTask struct {
	PatternCode DiagnosticPatternCode `json:"pattern_code"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Prompt      string                `json:"prompt"`
	Required    bool                  `json:"required"` // Whether this task is required (true) or optional (false)
}

// ProblemSet is a structured assignment generated from recurring patterns
type ProblemSet struct {
	ID                    string
	TutorialID            string
	OwnerSub              string
	WeekOf                time.Time
	AssignedFromSessionID string
	Status                string
	Tasks                 []ProblemSetTask
	ReviewNotes           string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ProblemSetPatternLink connects a problem set to the diagnostic entries it addresses
type ProblemSetPatternLink struct {
	ProblemSetID      string
	DiagnosticEntryID string
	PatternCode       string
}
