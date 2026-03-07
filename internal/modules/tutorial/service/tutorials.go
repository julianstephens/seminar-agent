package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/julianstephens/formation/internal/agent"
	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/modules/tutorial/repo"
	"github.com/julianstephens/formation/internal/observability"
	"github.com/julianstephens/formation/internal/sse"
)

// validDifficulties is the exhaustive set of allowed tutorial difficulty levels.
var validDifficulties = map[string]bool{
	"beginner":     true,
	"intermediate": true,
	"advanced":     true,
}

// ── TutorialService ───────────────────────────────────────────────────────────

// TutorialService implements all business operations for tutorials.
type TutorialService struct {
	repo *repo.TutorialRepo
}

// NewTutorialService constructs a TutorialService backed by the given repository.
func NewTutorialService(r *repo.TutorialRepo) *TutorialService {
	return &TutorialService{repo: r}
}

// ── Tutorial Create ────────────────────────────────────────────────────────────

// CreateTutorialParams holds all caller-supplied fields for creating a tutorial.
type CreateTutorialParams struct {
	Title       string
	Subject     string
	Description string
	Difficulty  string
}

// CreateTutorial validates params and persists a new tutorial owned by ownerSub.
func (s *TutorialService) CreateTutorial(
	ctx context.Context,
	ownerSub string,
	p CreateTutorialParams,
) (*domain.Tutorial, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, &ValidationError{Field: "title", Message: "must not be blank"}
	}
	if strings.TrimSpace(p.Subject) == "" {
		return nil, &ValidationError{Field: "subject", Message: "must not be blank"}
	}
	if p.Difficulty == "" {
		p.Difficulty = "beginner"
	}
	if !validDifficulties[p.Difficulty] {
		return nil, &ValidationError{Field: "difficulty", Message: "must be 'beginner', 'intermediate', or 'advanced'"}
	}

	tut := domain.Tutorial{
		Title:       p.Title,
		Subject:     p.Subject,
		Description: p.Description,
		Difficulty:  p.Difficulty,
	}
	return s.repo.CreateTutorial(ctx, ownerSub, tut)
}

// ── Tutorial Get ───────────────────────────────────────────────────────────────

// GetTutorial returns the tutorial with the given id if it is owned by ownerSub.
func (s *TutorialService) GetTutorial(ctx context.Context, id, ownerSub string) (*domain.Tutorial, error) {
	tut, err := s.repo.GetTutorialByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial", id)
	}
	return tut, nil
}

// ── Tutorial List ──────────────────────────────────────────────────────────────

// ListTutorials returns all tutorials owned by ownerSub.
func (s *TutorialService) ListTutorials(ctx context.Context, ownerSub string) ([]domain.Tutorial, error) {
	tutorials, err := s.repo.ListTutorials(ctx, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list tutorials: %w", err)
	}
	if tutorials == nil {
		tutorials = []domain.Tutorial{}
	}
	return tutorials, nil
}

// ── Tutorial Update ────────────────────────────────────────────────────────────

// UpdateTutorialParams holds all patchable tutorial fields; nil means "no change".
type UpdateTutorialParams struct {
	Title       *string
	Subject     *string
	Description *string
	Difficulty  *string
}

// UpdateTutorial applies a partial update to the tutorial and returns the updated record.
func (s *TutorialService) UpdateTutorial(
	ctx context.Context,
	id, ownerSub string,
	p UpdateTutorialParams,
) (*domain.Tutorial, error) {
	if p.Difficulty != nil && !validDifficulties[*p.Difficulty] {
		return nil, &ValidationError{Field: "difficulty", Message: "must be 'beginner', 'intermediate', or 'advanced'"}
	}
	patch := domain.TutorialPatch{
		Title:       p.Title,
		Subject:     p.Subject,
		Description: p.Description,
		Difficulty:  p.Difficulty,
	}
	tut, err := s.repo.UpdateTutorial(ctx, id, ownerSub, patch)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial", id)
	}
	return tut, nil
}

// ── Tutorial Delete ────────────────────────────────────────────────────────────

// DeleteTutorial removes the tutorial. Returns NotFoundError if it does not
// exist or is owned by a different user.
func (s *TutorialService) DeleteTutorial(ctx context.Context, id, ownerSub string) error {
	if err := s.repo.DeleteTutorial(ctx, id, ownerSub); err != nil {
		return wrapNotFound(err, "tutorial", id)
	}
	return nil
}

// ── TutorialSessionService ────────────────────────────────────────────────────

// TutorialSessionService implements all business operations for tutorial sessions.
type TutorialSessionService struct {
	sessions  *repo.TutorialRepo
	tutorials *repo.TutorialRepo
}

// NewTutorialSessionService constructs a TutorialSessionService backed by the given repository.
func NewTutorialSessionService(r *repo.TutorialRepo) *TutorialSessionService {
	return &TutorialSessionService{sessions: r, tutorials: r}
}

// ── Session Create ─────────────────────────────────────────────────────────────

// CreateTutorialSession creates a new in-progress tutorial session under tutorialID.
func (s *TutorialSessionService) CreateTutorialSession(
	ctx context.Context,
	ownerSub, tutorialID string,
	kind domain.TutorialSessionKind,
) (*domain.TutorialSession, error) {
	// Verify tutorial ownership.
	if _, err := s.tutorials.GetTutorialByID(ctx, tutorialID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial", tutorialID)
	}

	// Validate kind if provided (empty is allowed for backward compatibility).
	if kind != "" && !domain.ValidTutorialSessionKind(kind) {
		return nil, &ValidationError{Field: "kind", Message: "must be 'diagnostic' or 'extended'"}
	}

	sess := domain.TutorialSession{
		TutorialID: tutorialID,
		Kind:       kind,
	}
	created, err := s.sessions.CreateSession(ctx, ownerSub, sess)
	if err != nil {
		return nil, fmt.Errorf("create tutorial session: %w", err)
	}
	return created, nil
}

// ── Session Get ────────────────────────────────────────────────────────────────

// TutorialSessionDetail wraps a session with its artifacts and turns.
type TutorialSessionDetail struct {
	Session   *domain.TutorialSession
	Artifacts []domain.Artifact
	Turns     []domain.TutorialTurn
}

// GetTutorialSession returns the session, its artifacts, and turns if owned by ownerSub.
func (s *TutorialSessionService) GetTutorialSession(
	ctx context.Context,
	id, ownerSub string,
) (*TutorialSessionDetail, error) {
	sess, err := s.sessions.GetSessionByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", id)
	}

	artifacts, err := s.sessions.ListArtifactsBySessionID(ctx, id, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("get session artifacts: %w", err)
	}

	turns, err := s.sessions.ListTutorialTurns(ctx, id, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("get session turns: %w", err)
	}

	return &TutorialSessionDetail{Session: sess, Artifacts: artifacts, Turns: turns}, nil
}

// ── Session List ───────────────────────────────────────────────────────────────

// ListTutorialSessions returns all sessions for a tutorial in
// reverse-chronological order.
func (s *TutorialSessionService) ListTutorialSessions(
	ctx context.Context,
	tutorialID, ownerSub string,
) ([]domain.TutorialSession, error) {
	// Verify tutorial ownership.
	if _, err := s.tutorials.GetTutorialByID(ctx, tutorialID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial", tutorialID)
	}

	sessions, err := s.sessions.ListSessionsByTutorialID(ctx, tutorialID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list tutorial sessions: %w", err)
	}
	return sessions, nil
}

// ── Session Complete ───────────────────────────────────────────────────────────

// CompleteTutorialSession transitions an in-progress session to complete.
// Returns ErrSessionTerminalError if the session is already terminal.
func (s *TutorialSessionService) CompleteTutorialSession(
	ctx context.Context,
	id, ownerSub, notes string,
) (*domain.TutorialSession, error) {
	existing, err := s.sessions.GetSessionByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	if existing.IsTerminal() {
		return nil, &ErrSessionTerminalError{Status: string(existing.Status)}
	}

	sess, err := s.sessions.CompleteSession(ctx, id, ownerSub, notes)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	return sess, nil
}

// ── Session Abandon ────────────────────────────────────────────────────────────

// AbandonTutorialSession transitions an in-progress session to abandoned.
// Returns ErrSessionTerminalError if the session is already terminal.
func (s *TutorialSessionService) AbandonTutorialSession(
	ctx context.Context,
	id, ownerSub string,
) (*domain.TutorialSession, error) {
	existing, err := s.sessions.GetSessionByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	if existing.IsTerminal() {
		return nil, &ErrSessionTerminalError{Status: string(existing.Status)}
	}

	sess, err := s.sessions.AbandonSession(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	return sess, nil
}

// ── Session Delete ─────────────────────────────────────────────────────────────

// DeleteTutorialSession removes a session and all its associated artifacts.
func (s *TutorialSessionService) DeleteTutorialSession(ctx context.Context, id, ownerSub string) error {
	if err := s.sessions.DeleteSession(ctx, id, ownerSub); err != nil {
		return wrapNotFound(err, "tutorial_session", id)
	}
	return nil
}

// ── ArtifactService ───────────────────────────────────────────────────────────

// ArtifactService implements all business operations for artifacts.
type ArtifactService struct {
	repo *repo.TutorialRepo
}

// NewArtifactService constructs an ArtifactService backed by the given repository.
func NewArtifactService(r *repo.TutorialRepo) *ArtifactService {
	return &ArtifactService{repo: r}
}

// ── Artifact Create ────────────────────────────────────────────────────────────

// CreateArtifactParams holds all caller-supplied fields for creating an artifact.
type CreateArtifactParams struct {
	Kind    domain.ArtifactKind
	Title   string
	Content string
}

// CreateArtifact validates params and persists a new artifact under sessionID.
func (s *ArtifactService) CreateArtifact(
	ctx context.Context,
	ownerSub, sessionID string,
	p CreateArtifactParams,
) (*domain.Artifact, error) {
	if !domain.ValidArtifactKind(p.Kind) {
		return nil, &ValidationError{
			Field:   "kind",
			Message: "must be 'summary', 'notes', 'problem_set', or 'diagnostic'",
		}
	}
	if strings.TrimSpace(p.Title) == "" {
		return nil, &ValidationError{Field: "title", Message: "must not be blank"}
	}
	if strings.TrimSpace(p.Content) == "" {
		return nil, &ValidationError{Field: "content", Message: "must not be blank"}
	}

	// Verify session ownership.
	if _, err := s.repo.GetSessionByID(ctx, sessionID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	art := domain.Artifact{
		SessionID: sessionID,
		Kind:      p.Kind,
		Title:     p.Title,
		Content:   p.Content,
	}
	created, err := s.repo.CreateArtifact(ctx, ownerSub, art)
	if err != nil {
		return nil, fmt.Errorf("create artifact: %w", err)
	}
	return created, nil
}

// ── Artifact List ──────────────────────────────────────────────────────────────

// ListArtifacts returns all artifacts for a session.
func (s *ArtifactService) ListArtifacts(ctx context.Context, sessionID, ownerSub string) ([]domain.Artifact, error) {
	// Verify session ownership.
	if _, err := s.repo.GetSessionByID(ctx, sessionID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	artifacts, err := s.repo.ListArtifactsBySessionID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	return artifacts, nil
}

// ── Artifact Get ───────────────────────────────────────────────────────────────

// GetArtifact returns the artifact with the given id if owned by ownerSub.
func (s *ArtifactService) GetArtifact(ctx context.Context, id, ownerSub string) (*domain.Artifact, error) {
	art, err := s.repo.GetArtifactByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "artifact", id)
	}
	return art, nil
}

// ── Artifact Delete ────────────────────────────────────────────────────────────

// DeleteArtifact removes the artifact.
func (s *ArtifactService) DeleteArtifact(ctx context.Context, id, ownerSub string) error {
	if err := s.repo.DeleteArtifact(ctx, id, ownerSub); err != nil {
		return wrapNotFound(err, "artifact", id)
	}
	return nil
}

// ── TutorialTurnService ───────────────────────────────────────────────────────

// TutorialTurnService implements all business operations for tutorial session turns.
type TutorialTurnService struct {
	repo            *repo.TutorialRepo
	assembler       *agent.TutorialAssembler
	hub             *sse.Hub
	agent           agent.Provider // may be nil when no LLM is configured
	diagnosticSvc   *DiagnosticLedgerService
	enableStreaming bool
}

// NewTutorialTurnService constructs a TutorialTurnService backed by the given repository.
// provider may be nil; in that case the pipeline persists the user turn but skips agent response.
func NewTutorialTurnService(
	r *repo.TutorialRepo,
	assembler *agent.TutorialAssembler,
	hub *sse.Hub,
	provider agent.Provider,
	diagnosticSvc *DiagnosticLedgerService,
	enableStreaming bool,
) *TutorialTurnService {
	return &TutorialTurnService{
		repo:            r,
		assembler:       assembler,
		hub:             hub,
		agent:           provider,
		diagnosticSvc:   diagnosticSvc,
		enableStreaming: enableStreaming,
	}
}

// SubmitTutorialTurnResult holds the output from submitting a tutorial turn.
type SubmitTutorialTurnResult struct {
	UserTurn  *domain.TutorialTurn
	AgentTurn *domain.TutorialTurn
}

// SubmitTutorialTurn creates a user turn and may generate an agent response in the future.
// For now, this is a simplified version that only stores the user turn.
func (s *TutorialTurnService) SubmitTutorialTurn(
	ctx context.Context,
	sessionID, ownerSub, text string,
) (*SubmitTutorialTurnResult, error) {
	// Validate input.
	if strings.TrimSpace(text) == "" {
		return nil, &ValidationError{Field: "text", Message: "must not be blank"}
	}

	// Verify session exists and is owned by the user.
	sess, err := s.repo.GetSessionByID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	// Don't allow turns on terminal sessions.
	if sess.IsTerminal() {
		return nil, &ErrSessionTerminalError{Status: string(sess.Status)}
	}

	// Create the user turn.
	userTurn := domain.TutorialTurn{
		SessionID: sessionID,
		Speaker:   "user",
		Text:      text,
	}
	created, err := s.repo.CreateTutorialTurn(ctx, sessionID, ownerSub, userTurn)
	if err != nil {
		return nil, fmt.Errorf("create user turn: %w", err)
	}

	// Emit turn_added SSE for user turn.
	s.hub.PublishTutorialTurnAdded(created)

	result := &SubmitTutorialTurnResult{
		UserTurn:  created,
		AgentTurn: nil,
	}

	// If no agent is configured, return just the user turn.
	if s.agent == nil || s.assembler == nil {
		return result, nil
	}

	// Load tutorial and artifacts for context.
	tutorial, err := s.repo.GetTutorialByID(ctx, sess.TutorialID, ownerSub)
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to load tutorial for prompt assembly",
			"session_id", sessionID,
			"tutorial_id", sess.TutorialID,
			"error", err.Error(),
		)
		return result, nil // Return user turn even if we can't load tutorial
	}

	// Load prior turns for conversation history.
	priorTurns, err := s.repo.ListTutorialTurns(ctx, sessionID, ownerSub)
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to load prior turns for prompt assembly",
			"session_id", sessionID,
			"error", err.Error(),
		)
		return result, nil
	}

	// Load artifacts to include in the prompt.
	artifacts, err := s.repo.ListArtifactsBySessionID(ctx, sessionID, ownerSub)
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to load artifacts for prompt assembly",
			"session_id", sessionID,
			"error", err.Error(),
		)
		artifacts = []domain.Artifact{} // Continue with empty artifacts
	}

	// Format artifacts for the prompt.
	artifactsText := formatArtifacts(artifacts)

	// ── Diagnostic Ledger Integration ──

	// Determine session kind and task mode
	sessionKind := string(sess.Kind)
	if sessionKind == "" {
		sessionKind = "diagnostic" // Default to diagnostic
	}

	// Determine task mode based on session kind and presence of prior problem set
	taskMode := "review_only"
	weekOf := sess.StartedAt.Truncate(24 * time.Hour)

	var previousProblemSet *domain.ProblemSet
	var priorDiagnosticsSummary string
	var problemSetResponseText string

	// Load previous problem set if this is an extended session
	if sess.Kind == domain.TutorialSessionKindExtended {
		previousProblemSet, err = s.diagnosticSvc.GetPreviousProblemSet(ctx, sess.TutorialID, ownerSub, weekOf)
		if err != nil {
			logger := observability.LoggerFromContext(ctx)
			logger.Error("failed to load previous problem set",
				"session_id", sessionID,
				"error", err.Error(),
			)
		}

		if previousProblemSet != nil {
			taskMode = "problemset_review"

			// Look for problem set response artifact
			for _, art := range artifacts {
				if art.Kind == domain.ArtifactKindProblemSetResponse {
					break
				}
			}
		} else {
			// No previous problem set, so this is a generation session
			taskMode = "problemset_generation"
		}
	}

	// Load prior diagnostics summary
	patternSummary, err := s.diagnosticSvc.BuildPatternSummary(ctx, sess.TutorialID, ownerSub, 4)
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to build pattern summary",
			"session_id", sessionID,
			"error", err.Error(),
		)
	} else {
		priorDiagnosticsSummary = formatPatternSummary(patternSummary)
	}

	// Load current week diagnostics
	weekSummary, err := s.diagnosticSvc.BuildCurrentWeekSummary(ctx, sess.TutorialID, ownerSub, weekOf)
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to build week summary",
			"session_id", sessionID,
			"error", err.Error(),
		)
	} else {
		// Append current week details to prior diagnostics summary
		if len(weekSummary.Entries) > 0 {
			priorDiagnosticsSummary += formatWeekSummary(weekSummary)
		}
	}

	// Format previous problem set if present
	previousProblemSetText := ""
	if previousProblemSet != nil {
		previousProblemSetText = formatProblemSet(*previousProblemSet)
	}

	// Assemble the prompt.
	messages, err := s.assembler.AssembleTutorial(agent.TutorialAssembleParams{
		TutorialTitle:      tutorial.Title,
		SessionKind:        sessionKind,
		TaskMode:           taskMode,
		WeekOf:             weekOf.Format("2006-01-02"),
		Artifacts:          artifactsText,
		PriorDiagnostics:   priorDiagnosticsSummary,
		PreviousProblemSet: previousProblemSetText,
		ProblemSetResponse: problemSetResponseText,
		Turns:              priorTurns,
	})
	if err != nil {
		logger := observability.LoggerFromContext(ctx)
		logger.Error("failed to assemble tutorial prompt",
			"session_id", sessionID,
			"error", err.Error(),
		)
		return result, nil
	}

	// Create an empty agent turn first to get a turn ID for streaming.
	agentTurn := domain.TutorialTurn{
		SessionID: sessionID,
		Speaker:   "agent",
		Text:      "", // Will be updated after streaming completes
	}
	agentCreated, err := s.repo.CreateTutorialTurn(ctx, sessionID, ownerSub, agentTurn)
	if err != nil {
		return result, fmt.Errorf("create agent turn: %w", err)
	}

	// Emit initial turn_added SSE for agent turn.
	s.hub.PublishTutorialTurnAdded(agentCreated)

	// Call the agent with or without streaming.
	var agentText string
	logger := observability.LoggerFromContext(ctx)

	if s.enableStreaming {
		logger.Info("starting agent streaming",
			"session_id", sessionID,
			"turn_id", agentCreated.ID,
		)
		// Use streaming mode: publish chunks as they arrive.
		chunkCount := 0
		chunkFn := func(chunk string) error {
			chunkCount++
			logger.Debug("publishing chunk",
				"session_id", sessionID,
				"turn_id", agentCreated.ID,
				"chunk_num", chunkCount,
				"chunk_len", len(chunk),
			)
			s.hub.PublishAgentResponseChunk(sessionID, agentCreated.ID, chunk, false)
			return nil
		}
		agentText, err = s.agent.CompleteStream(ctx, messages, chunkFn)
		logger.Info("agent streaming completed",
			"session_id", sessionID,
			"turn_id", agentCreated.ID,
			"total_chunks", chunkCount,
			"total_length", len(agentText),
		)
		if err != nil {
			logger.Error("agent streaming call failed",
				"session_id", sessionID,
				"turn_id", agentCreated.ID,
				"error", err.Error(),
			)
			return result, nil // Return user turn even if agent call fails
		}
	} else {
		// Use non-streaming mode.
		agentText, err = s.agent.Complete(ctx, messages)
		if err != nil {
			logger.Error("agent call failed",
				"session_id", sessionID,
				"turn_id", agentCreated.ID,
				"error", err.Error(),
			)
			return result, nil // Return user turn even if agent call fails
		}
	}

	// ── Parse and persist diagnostic entries ──

	diagnosticInputs, err := ParseDiagnosticJSON(agentText)
	if err != nil {
		logger.Error("failed to parse diagnostic JSON block",
			"session_id", sessionID,
			"error", err.Error(),
		)
		// Continue without diagnostics rather than failing
	} else if len(diagnosticInputs) > 0 {
		diagnosticEntries, err := ConvertToDiagnosticEntries(diagnosticInputs)
		if err != nil {
			logger.Error("failed to convert diagnostic entries",
				"session_id", sessionID,
				"error", err.Error(),
			)
		} else {
			err = s.diagnosticSvc.RecordEntries(ctx, ownerSub, sess.TutorialID, sessionID, weekOf, diagnosticEntries)
			if err != nil {
				logger.Error("failed to record diagnostic entries",
					"session_id", sessionID,
					"error", err.Error(),
				)
			}
		}
	}

	// Strip the diagnostic JSON block from the agent response before storing
	agentTextStripped := StripDiagnosticBlock(agentText)

	// Update pattern statuses after extended review
	if sess.Kind == domain.TutorialSessionKindExtended {
		err = s.diagnosticSvc.UpdatePatternStatuses(ctx, sess.TutorialID, ownerSub, weekOf)
		if err != nil {
			logger.Error("failed to update pattern statuses",
				"session_id", sessionID,
				"error", err.Error(),
			)
		}
	}

	// Update the agent turn with the complete stripped text.
	agentCreated, err = s.repo.UpdateTutorialTurn(ctx, agentCreated.ID, sessionID, ownerSub, agentTextStripped)
	if err != nil {
		return result, fmt.Errorf("update agent turn: %w", err)
	}

	// Emit final chunk with complete text if streaming was enabled.
	if s.enableStreaming {
		logger.Info("sending final stream chunk",
			"session_id", sessionID,
			"turn_id", agentCreated.ID,
		)
		s.hub.PublishAgentResponseChunk(sessionID, agentCreated.ID, "", true)
	}

	result.AgentTurn = agentCreated
	return result, nil
}

// ListTutorialTurns returns all turns for a session in chronological order.
func (s *TutorialTurnService) ListTutorialTurns(
	ctx context.Context,
	sessionID, ownerSub string,
) ([]domain.TutorialTurn, error) {
	// Verify session ownership.
	if _, err := s.repo.GetSessionByID(ctx, sessionID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	turns, err := s.repo.ListTutorialTurns(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list tutorial turns: %w", err)
	}
	return turns, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

// formatArtifacts converts a list of artifacts into a formatted string for the prompt.
func formatArtifacts(artifacts []domain.Artifact) string {
	if len(artifacts) == 0 {
		return "(No artifacts submitted yet)"
	}

	var sb strings.Builder
	for i, art := range artifacts {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("--- %s: %s ---\n", art.Kind, art.Title))
		sb.WriteString(art.Content)
	}
	return sb.String()
}

// formatPatternSummary converts a pattern summary into prompt-ready text.
func formatPatternSummary(summary PatternSummary) string {
	if len(summary.Items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("PATTERN LEDGER\n\n")
	sb.WriteString("Recurring patterns across the last 4 weeks:\n")

	for _, item := range summary.Items {
		sb.WriteString(fmt.Sprintf("- %s — %d occurrences — %s (last seen: %s)\n",
			item.PatternCode, item.Occurrences, item.Trend, item.LastSeenWeek))
	}

	return sb.String()
}

// formatWeekSummary appends current week diagnostic details to a pattern summary.
func formatWeekSummary(summary WeekSummary) string {
	if len(summary.Entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nCurrent week diagnostics:\n")

	for _, entry := range summary.Entries {
		sb.WriteString(fmt.Sprintf("- %s (severity %d) observed", entry.PatternCode, entry.Severity))
		if len(entry.Evidence) > 0 {
			sb.WriteString(fmt.Sprintf(" in %s", entry.Evidence[0].ArtifactTitle))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatProblemSet converts a problem set into prompt-ready text.
func formatProblemSet(ps domain.ProblemSet) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PREVIOUS PROBLEM SET (Week of %s)\n\n", ps.WeekOf.Format("2006-01-02")))

	for i, task := range ps.Tasks {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, task.Title))
		sb.WriteString(fmt.Sprintf("   Pattern: %s\n", task.PatternCode))
		if task.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", task.Description))
		}
		if i < len(ps.Tasks)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
