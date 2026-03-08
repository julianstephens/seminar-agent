package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/julianstephens/formation/internal/domain"
)

// ProblemSetBlock represents the structured problem set data in an agent response.
type ProblemSetBlock struct {
	Tasks []ProblemSetTaskInput `json:"tasks"`
}

// ProblemSetTaskInput is the shape expected from the agent's JSON block.
type ProblemSetTaskInput struct {
	PatternCode string `json:"pattern_code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Required    bool   `json:"required"` // Whether this task is required completion
}

// ParseProblemSetJSON extracts and parses the [PROBLEMSET_JSON]...[/PROBLEMSET_JSON] block
// from an agent response. Returns an empty slice if no block is found.
func ParseProblemSetJSON(agentResponse string) ([]ProblemSetTaskInput, error) {
	// Find the problem set JSON block using regex
	pattern := regexp.MustCompile(`(?s)\[PROBLEMSET_JSON\]\s*(.*?)\s*\[/PROBLEMSET_JSON\]`)
	matches := pattern.FindStringSubmatch(agentResponse)

	if len(matches) < 2 {
		// No problem set block found, return empty slice
		return []ProblemSetTaskInput{}, nil
	}

	jsonStr := strings.TrimSpace(matches[1])

	var block ProblemSetBlock
	if err := json.Unmarshal([]byte(jsonStr), &block); err != nil {
		return nil, fmt.Errorf("parse problem set json: %w", err)
	}

	return block.Tasks, nil
}

// ConvertToProblemSetTasks converts parsed input tasks into domain.ProblemSetTask objects.
// This validates the pattern codes and validates the structure.
func ConvertToProblemSetTasks(inputs []ProblemSetTaskInput) ([]domain.ProblemSetTask, error) {
	// Validate problem set structure first
	if err := ValidateProblemSetStructure(inputs); err != nil {
		return nil, err
	}

	var tasks []domain.ProblemSetTask

	for i, input := range inputs {
		// Validate pattern code
		patternCode := domain.DiagnosticPatternCode(input.PatternCode)
		if !isValidProblemSetPatternCode(patternCode) {
			return nil, fmt.Errorf("invalid pattern code at task %d: %s", i, input.PatternCode)
		}

		// Validate required fields
		if input.Title == "" {
			return nil, fmt.Errorf("missing title at task %d", i)
		}
		if input.Description == "" {
			return nil, fmt.Errorf("missing description at task %d", i)
		}
		if input.Prompt == "" {
			return nil, fmt.Errorf("missing prompt at task %d", i)
		}

		task := domain.ProblemSetTask{
			PatternCode: patternCode,
			Title:       input.Title,
			Description: input.Description,
			Prompt:      input.Prompt,
			Required:    input.Required,
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ValidateProblemSetStructure validates that a problem set has the correct structure:
// - Exactly 10 tasks
// - 3 to 5 required tasks
func ValidateProblemSetStructure(inputs []ProblemSetTaskInput) error {
	// Check for exactly 10 tasks
	if len(inputs) != 10 {
		return fmt.Errorf("problem set must have exactly 10 tasks, got %d", len(inputs))
	}

	// Count required tasks
	requiredCount := 0
	for _, input := range inputs {
		if input.Required {
			requiredCount++
		}
	}

	// Check for 3-5 required tasks
	if requiredCount < 3 || requiredCount > 5 {
		return fmt.Errorf("problem set must have 3 to 5 required tasks, got %d", requiredCount)
	}

	return nil
}

// StripProblemSetBlock removes the problem set JSON block from the agent response,
// leaving only the student-facing text.
func StripProblemSetBlock(agentResponse string) string {
	pattern := regexp.MustCompile(`(?s)\[PROBLEMSET_JSON\].*?\[/PROBLEMSET_JSON\]\s*`)
	return strings.TrimSpace(pattern.ReplaceAllString(agentResponse, ""))
}

// ── validation helpers ────────────────────────────────────────────────────────

func isValidProblemSetPatternCode(code domain.DiagnosticPatternCode) bool {
	// Accept any non-empty pattern code in UPPER_SNAKE_CASE format
	// This allows for flexibility as new patterns are added over time
	codeStr := string(code)
	if codeStr == "" {
		return false
	}
	// Basic validation: must contain only uppercase letters and underscores
	for _, ch := range codeStr {
		if !((ch >= 'A' && ch <= 'Z') || ch == '_') {
			return false
		}
	}
	return true
}
