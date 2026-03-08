package service

import (
	"testing"
)

// ── ValidateProblemSetStructure ───────────────────────────────────────────────

func TestValidateProblemSetStructure_valid10Tasks3Required(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 3, // First 3 are required
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err != nil {
		t.Fatalf("expected no error for 10 tasks with 3 required, got: %v", err)
	}
}

func TestValidateProblemSetStructure_valid10Tasks5Required(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 5, // First 5 are required
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err != nil {
		t.Fatalf("expected no error for 10 tasks with 5 required, got: %v", err)
	}
}

func TestValidateProblemSetStructure_invalid9Tasks(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 9)
	for i := 0; i < 9; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 3,
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err == nil {
		t.Fatal("expected error for 9 tasks, got nil")
	}
}

func TestValidateProblemSetStructure_invalid11Tasks(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 11)
	for i := 0; i < 11; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 3,
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err == nil {
		t.Fatal("expected error for 11 tasks, got nil")
	}
}

func TestValidateProblemSetStructure_invalid2Required(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 2, // Only 2 required
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err == nil {
		t.Fatal("expected error for 2 required tasks, got nil")
	}
}

func TestValidateProblemSetStructure_invalid6Required(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    i < 6, // 6 required
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err == nil {
		t.Fatal("expected error for 6 required tasks, got nil")
	}
}

func TestValidateProblemSetStructure_invalid0Required(t *testing.T) {
	t.Parallel()
	tasks := make([]ProblemSetTaskInput, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = ProblemSetTaskInput{
			PatternCode: "TEXT_DRIFT",
			Title:       "Task",
			Description: "Description",
			Prompt:      "Prompt",
			Required:    false, // None required
		}
	}

	err := ValidateProblemSetStructure(tasks)
	if err == nil {
		t.Fatal("expected error for 0 required tasks, got nil")
	}
}
