package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/julianstephens/formation/internal/domain"
)

// DiagnosticBlock represents the structured diagnostic data in an agent response.
type DiagnosticBlock struct {
	Entries []DiagnosticEntryInput `json:"entries"`
}

// DiagnosticEntryInput is the shape expected from the agent's JSON block.
type DiagnosticEntryInput struct {
	PatternCode string                      `json:"pattern_code"`
	Severity    int                         `json:"severity"`
	Status      string                      `json:"status"`
	Evidence    []domain.DiagnosticEvidence `json:"evidence"`
	Notes       string                      `json:"notes"`
}

// ParseDiagnosticJSON extracts and parses the [DIAGNOSTIC_JSON]...[/DIAGNOSTIC_JSON] block
// from an agent response. Returns an empty slice if no block is found.
func ParseDiagnosticJSON(agentResponse string) ([]DiagnosticEntryInput, error) {
	// Find the diagnostic JSON block using regex
	pattern := regexp.MustCompile(`(?s)\[DIAGNOSTIC_JSON\]\s*(.*?)\s*\[/DIAGNOSTIC_JSON\]`)
	matches := pattern.FindStringSubmatch(agentResponse)

	if len(matches) < 2 {
		// No diagnostic block found, return empty slice
		return []DiagnosticEntryInput{}, nil
	}

	jsonStr := strings.TrimSpace(matches[1])

	var block DiagnosticBlock
	if err := json.Unmarshal([]byte(jsonStr), &block); err != nil {
		return nil, fmt.Errorf("parse diagnostic json: %w", err)
	}

	return block.Entries, nil
}

// ConvertToDiagnosticEntries converts parsed input entries into domain.DiagnosticEntry objects.
// This does basic validation of the pattern codes and status values.
func ConvertToDiagnosticEntries(inputs []DiagnosticEntryInput) ([]domain.DiagnosticEntry, error) {
	var entries []domain.DiagnosticEntry

	for i, input := range inputs {
		// Validate pattern code
		patternCode := domain.DiagnosticPatternCode(input.PatternCode)
		if !isValidPatternCode(patternCode) {
			return nil, fmt.Errorf("invalid pattern code at entry %d: %s", i, input.PatternCode)
		}

		// Validate status
		status := domain.DiagnosticStatus(input.Status)
		if !isValidStatus(status) {
			return nil, fmt.Errorf("invalid status at entry %d: %s", i, input.Status)
		}

		// Validate severity
		if input.Severity < 1 || input.Severity > 5 {
			return nil, fmt.Errorf("invalid severity at entry %d: must be between 1 and 5, got %d", i, input.Severity)
		}

		entry := domain.DiagnosticEntry{
			PatternCode: patternCode,
			Severity:    input.Severity,
			Status:      status,
			Evidence:    input.Evidence,
			Notes:       input.Notes,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// StripDiagnosticBlock removes the diagnostic JSON block from the agent response,
// leaving only the student-facing text.
func StripDiagnosticBlock(agentResponse string) string {
	pattern := regexp.MustCompile(`(?s)\[DIAGNOSTIC_JSON\].*?\[/DIAGNOSTIC_JSON\]\s*`)
	return strings.TrimSpace(pattern.ReplaceAllString(agentResponse, ""))
}

// ── validation helpers ────────────────────────────────────────────────────────

func isValidPatternCode(code domain.DiagnosticPatternCode) bool {
	switch code {
	case domain.PatternUndefinedTerms,
		domain.PatternTextDrift,
		domain.PatternHiddenPremises,
		domain.PatternWeakStructure,
		domain.PatternRhetoricalInflation,
		domain.PatternPrematureSynthesis:
		return true
	default:
		return false
	}
}

func isValidStatus(status domain.DiagnosticStatus) bool {
	switch status {
	case domain.DiagnosticActive,
		domain.DiagnosticImproving,
		domain.DiagnosticResolved:
		return true
	default:
		return false
	}
}
