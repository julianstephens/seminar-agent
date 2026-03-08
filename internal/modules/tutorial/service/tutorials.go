package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/julianstephens/formation/internal/agent"
	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/modules/tutorial/repo"
	"github.com/julianstephens/formation/internal/observability"
	sharedRepo "github.com/julianstephens/formation/internal/repo"
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
	logger := observability.LoggerFromContext(ctx)
	logger.Debug("creating tutorial",
		slog.String("owner", ownerSub),
		slog.String("title", p.Title),
		slog.String("difficulty", p.Difficulty),
	)

	if strings.TrimSpace(p.Title) == "" {
		logger.Debug("validation failed: blank title")
		return nil, &ValidationError{Field: "title", Message: "must not be blank"}
	}
	if strings.TrimSpace(p.Subject) == "" {
		logger.Debug("validation failed: blank subject")
		return nil, &ValidationError{Field: "subject", Message: "must not be blank"}
	}
	if p.Difficulty == "" {
		p.Difficulty = "beginner"
	}
	if !validDifficulties[p.Difficulty] {
		logger.Debug("invalid difficulty", slog.String("difficulty", p.Difficulty))
		return nil, &ValidationError{Field: "difficulty", Message: "must be 'beginner', 'intermediate', or 'advanced'"}
	}

	tut := domain.Tutorial{
		Title:       p.Title,
		Subject:     p.Subject,
		Description: p.Description,
		Difficulty:  p.Difficulty,
	}
	created, err := s.repo.CreateTutorial(ctx, ownerSub, tut)
	if err != nil {
		logger.Error("failed to create tutorial", slog.String("error", err.Error()))
		return nil, err
	}
	logger.Debug("tutorial created", slog.String("id", created.ID))
	return created, nil
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
	logger := observability.LoggerFromContext(ctx)
	logger.Debug("creating tutorial session",
		slog.String("owner", ownerSub),
		slog.String("tutorial_id", tutorialID),
		slog.String("kind", string(kind)),
	)

	// Verify tutorial ownership.
	if _, err := s.tutorials.GetTutorialByID(ctx, tutorialID, ownerSub); err != nil {
		logger.Debug("tutorial not found", slog.String("tutorial_id", tutorialID), slog.String("error", err.Error()))
		return nil, wrapNotFound(err, "tutorial", tutorialID)
	}

	// Validate kind if provided (empty is allowed for backward compatibility).
	if kind != "" && !domain.ValidTutorialSessionKind(kind) {
		logger.Debug("invalid session kind", slog.String("kind", string(kind)))
		return nil, &ValidationError{Field: "kind", Message: "must be 'diagnostic' or 'extended'"}
	}

	sess := domain.TutorialSession{
		TutorialID: tutorialID,
		Kind:       kind,
	}
	created, err := s.sessions.CreateSession(ctx, ownerSub, sess)
	if err != nil {
		logger.Error("failed to create tutorial session", slog.String("error", err.Error()))
		return nil, fmt.Errorf("create tutorial session: %w", err)
	}
	logger.Debug("tutorial session created", slog.String("id", created.ID))
	return created, nil
}

// ── Session Get ────────────────────────────────────────────────────────────────

// TutorialSessionDetail wraps a session with its artifacts and turns.
type TutorialSessionDetail struct {
	Session    *domain.TutorialSession
	Artifacts  []domain.Artifact
	Turns      []domain.TutorialTurn
	ProblemSet *domain.ProblemSet
}

// GetTutorialSession returns the session, its artifacts, and turns if owned by ownerSub.
func (s *TutorialSessionService) GetTutorialSession(
	ctx context.Context,
	id, ownerSub string,
) (*TutorialSessionDetail, error) {
	logger := observability.LoggerFromContext(ctx)
	logger.Debug("fetching tutorial session", slog.String("id", id), slog.String("owner", ownerSub))

	sess, err := s.sessions.GetSessionByID(ctx, id, ownerSub)
	if err != nil {
		logger.Debug("tutorial session not found", slog.String("id", id), slog.String("error", err.Error()))
		return nil, wrapNotFound(err, "tutorial_session", id)
	}

	artifacts, err := s.sessions.ListArtifactsBySessionID(ctx, id, ownerSub)
	if err != nil {
		logger.Error("failed to fetch session artifacts", slog.String("id", id), slog.String("error", err.Error()))
		return nil, fmt.Errorf("get session artifacts: %w", err)
	}

	turns, err := s.sessions.ListTutorialTurns(ctx, id, ownerSub)
	if err != nil {
		logger.Error("failed to fetch session turns", slog.String("id", id), slog.String("error", err.Error()))
		return nil, fmt.Errorf("get session turns: %w", err)
	}

	// Fetch problem set if one is assigned to this session
	var problemSet *domain.ProblemSet
	ps, err := s.sessions.GetProblemSetBySession(ctx, id, ownerSub)
	if err == nil {
		problemSet = ps
		logger.Debug("problem set found for session", slog.String("problem_set_id", ps.ID))
	} else {
		logger.Debug("no problem set found for session", slog.String("error", err.Error()))
	}
	// Ignore not found error - it's valid for sessions to have no problem set

	logger.Debug("tutorial session fetched",
		slog.String("id", id),
		slog.Int("artifact_count", len(artifacts)),
		slog.Int("turn_count", len(turns)),
	)
	return &TutorialSessionDetail{Session: sess, Artifacts: artifacts, Turns: turns, ProblemSet: problemSet}, nil
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
	logger := observability.LoggerFromContext(ctx)
	logger.Debug("completing tutorial session", slog.String("id", id), slog.String("owner", ownerSub))

	existing, err := s.sessions.GetSessionByID(ctx, id, ownerSub)
	if err != nil {
		logger.Debug("session not found", slog.String("id", id), slog.String("error", err.Error()))
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	if existing.IsTerminal() {
		logger.Debug("session already terminal", slog.String("id", id), slog.String("status", string(existing.Status)))
		return nil, &ErrSessionTerminalError{Status: string(existing.Status)}
	}

	sess, err := s.sessions.CompleteSession(ctx, id, ownerSub, notes)
	if err != nil {
		logger.Error("failed to complete session", slog.String("id", id), slog.String("error", err.Error()))
		return nil, wrapNotFound(err, "tutorial_session", id)
	}
	logger.Debug("tutorial session completed", slog.String("id", id))
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

// ── Problem Set Methods ────────────────────────────────────────────────────────

// GetSessionProblemSet returns the problem set assigned from a specific session.
func (s *TutorialSessionService) GetSessionProblemSet(
	ctx context.Context,
	sessionID, ownerSub string,
) (*domain.ProblemSet, error) {
	// Verify session ownership first
	if _, err := s.sessions.GetSessionByID(ctx, sessionID, ownerSub); err != nil {
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	ps, err := s.sessions.GetProblemSetBySession(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "problem_set", "for session "+sessionID)
	}
	return ps, nil
}

// DeleteSessionProblemSet soft deletes a problem set by updating its status to 'deleted'.
func (s *TutorialSessionService) DeleteSessionProblemSet(
	ctx context.Context,
	problemSetID, ownerSub string,
) error {
	_, err := s.sessions.UpdateProblemSetStatus(ctx, problemSetID, ownerSub, "deleted")
	if err != nil {
		return wrapNotFound(err, "problem_set", problemSetID)
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
	Kind         domain.ArtifactKind
	Title        string
	Content      string
	ProblemSetID string
}

// CreateArtifact validates params and persists a new artifact under sessionID.
func (s *ArtifactService) CreateArtifact(
	ctx context.Context,
	ownerSub, sessionID string,
	p CreateArtifactParams,
) (*domain.Artifact, error) {
	logger := observability.LoggerFromContext(ctx)
	logger.Debug("creating artifact",
		slog.String("owner", ownerSub),
		slog.String("session_id", sessionID),
		slog.String("kind", string(p.Kind)),
		slog.String("title", p.Title),
	)

	if !domain.ValidArtifactKind(p.Kind) {
		logger.Debug("invalid artifact kind", slog.String("kind", string(p.Kind)))
		return nil, &ValidationError{
			Field:   "kind",
			Message: "must be 'summary', 'notes', 'problem_set', or 'diagnostic'",
		}
	}
	if strings.TrimSpace(p.Title) == "" {
		logger.Debug("validation failed: blank title")
		return nil, &ValidationError{Field: "title", Message: "must not be blank"}
	}
	if strings.TrimSpace(p.Content) == "" {
		logger.Debug("validation failed: blank content")
		return nil, &ValidationError{Field: "content", Message: "must not be blank"}
	}

	// Verify session ownership.
	if _, err := s.repo.GetSessionByID(ctx, sessionID, ownerSub); err != nil {
		logger.Debug("session not found", slog.String("session_id", sessionID), slog.String("error", err.Error()))
		return nil, wrapNotFound(err, "tutorial_session", sessionID)
	}

	art := domain.Artifact{
		SessionID:    sessionID,
		Kind:         p.Kind,
		Title:        p.Title,
		Content:      p.Content,
		ProblemSetID: p.ProblemSetID,
	}
	created, err := s.repo.CreateArtifact(ctx, ownerSub, art)
	if err != nil {
		logger.Error("failed to create artifact", slog.String("error", err.Error()))
		return nil, fmt.Errorf("create artifact: %w", err)
	}
	logger.Debug("artifact created", slog.String("id", created.ID))
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

	// Parse and validate slash commands before storing the turn.
	cmd, err := parseAndValidateTutorialCommand(text, sess)
	if err != nil {
		return nil, err
	}

	// Parse command options if this is a /problem-set command.
	// We'll use these options throughout the function.
	var cmdOpts *problemSetCommandOptions
	if cmd == tutorialCommandProblemSet {
		opts, err := parseProblemSetCommandOptions(text)
		if err != nil {
			return nil, err
		}
		cmdOpts = &opts

		// Ensure no problem set already exists for this session (only for commit mode).
		if cmdOpts.Mode == "commit" {
			existing, psErr := s.repo.GetProblemSetBySession(ctx, sessionID, ownerSub)
			if psErr == nil && existing != nil {
				return nil, &ValidationError{
					Field:   "text",
					Message: "a problem set already exists for this session; delete it before generating a new one",
				}
			}
		}
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
		logger := observability.LoggerFromContext(ctx)
		logger.Warn("agent not configured, skipping agent response",
			"session_id", sessionID,
		)
		// Mark the user turn as failed
		if _, markErr := s.repo.MarkTutorialTurnFailed(ctx, created.ID, sessionID, ownerSub); markErr != nil {
			logger.Error("failed to mark user turn as failed",
				"session_id", sessionID,
				"turn_id", created.ID,
				"error", markErr.Error(),
			)
		}
		// Publish error to UI via SSE
		s.hub.PublishError(sessionID, "Agent is not configured. Please check server configuration.")
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

	// week_of is always the Sunday of the week in which the session started.
	weekOf := sundayOfWeek(sess.StartedAt)

	// Determine task mode.
	// A /problem-set command always forces generation mode; otherwise auto-detect.
	taskMode := "review_only"

	var previousProblemSet *domain.ProblemSet
	var priorDiagnosticsSummary string
	var problemSetResponseText string

	if cmd == tutorialCommandProblemSet {
		// Explicit command: always generate a new problem set.
		taskMode = "problemset_generation"
	} else if sess.Kind == domain.TutorialSessionKindExtended {
		// Load previous problem set if this is an extended session
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

	// Determine difficulty level for the prompt.
	// If this is a /problem-set command, use the command's difficulty; otherwise use tutorial difficulty.
	promptDifficulty := tutorial.Difficulty
	if cmdOpts != nil {
		promptDifficulty = mapCommandDifficultyToPromptDifficulty(cmdOpts.Difficulty)
	}

	// Assemble the prompt.
	messages, err := s.assembler.AssembleTutorial(agent.TutorialAssembleParams{
		TutorialTitle:      tutorial.Title,
		SessionKind:        sessionKind,
		TaskMode:           taskMode,
		Difficulty:         promptDifficulty,
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

	// Note: We emit turn_added SSE for agent turn only after confirming agent call succeeds

	// Call the agent with or without streaming.
	var agentText string
	logger := observability.LoggerFromContext(ctx)

	if s.enableStreaming {
		logger.Info("starting agent streaming",
			"session_id", sessionID,
			"turn_id", agentCreated.ID,
		)

		// Emit initial turn_added SSE for agent turn before streaming starts.
		s.hub.PublishTutorialTurnAdded(agentCreated)

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
			// Delete the empty agent turn
			if delErr := s.repo.DeleteTutorialTurn(ctx, agentCreated.ID, sessionID, ownerSub); delErr != nil {
				logger.Error("failed to delete empty agent turn",
					"session_id", sessionID,
					"turn_id", agentCreated.ID,
					"error", delErr.Error(),
				)
			}
			// Mark the user turn as failed
			if _, markErr := s.repo.MarkTutorialTurnFailed(ctx, created.ID, sessionID, ownerSub); markErr != nil {
				logger.Error("failed to mark user turn as failed",
					"session_id", sessionID,
					"turn_id", created.ID,
					"error", markErr.Error(),
				)
			}
			// Publish error to UI via SSE
			s.hub.PublishError(sessionID, fmt.Sprintf("Agent call failed: %v", err))
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
			// Delete the empty agent turn
			if delErr := s.repo.DeleteTutorialTurn(ctx, agentCreated.ID, sessionID, ownerSub); delErr != nil {
				logger.Error("failed to delete empty agent turn",
					"session_id", sessionID,
					"turn_id", agentCreated.ID,
					"error", delErr.Error(),
				)
			}
			// Mark the user turn as failed
			if _, markErr := s.repo.MarkTutorialTurnFailed(ctx, created.ID, sessionID, ownerSub); markErr != nil {
				logger.Error("failed to mark user turn as failed",
					"session_id", sessionID,
					"turn_id", created.ID,
					"error", markErr.Error(),
				)
			}
			// Publish error to UI via SSE
			s.hub.PublishError(sessionID, fmt.Sprintf("Agent call failed: %v", err))
			return result, nil // Return user turn even if agent call fails
		}

		// Emit turn_added SSE for agent turn now that we have the response.
		s.hub.PublishTutorialTurnAdded(agentCreated)
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

	// ── Parse and persist problem set (extended sessions only) ──

	if sess.Kind == domain.TutorialSessionKindExtended {
		logger.Info("attempting to parse problem set from agent response",
			"session_id", sessionID,
			"task_mode", taskMode,
			"is_problemset_command", cmd == tutorialCommandProblemSet,
		)
		problemSetInputs, err := ParseProblemSetJSON(agentText)
		if err != nil {
			logger.Error("failed to parse problem set JSON block",
				"session_id", sessionID,
				"error", err.Error(),
			)
			// Continue without problem set rather than failing
		} else if len(problemSetInputs) == 0 {
			// No problem set found in agent response
			// Log warning if this was a /problem-set command (expected generation)
			if cmd == tutorialCommandProblemSet {
				logger.Warn("problem set command was issued but agent response contains no [PROBLEMSET_JSON] block",
					"session_id", sessionID,
					"task_mode", taskMode,
				)
			}
		} else if len(problemSetInputs) > 0 {
			logger.Info("parsed problem set from agent response",
				"session_id", sessionID,
				"task_count", len(problemSetInputs),
			)
			// Check if mode is preview - if so, skip persistence
			if cmdOpts != nil && cmdOpts.Mode == "preview" {
				logger.Info("problem set generated in preview mode, skipping persistence",
					"session_id", sessionID,
					"task_count", len(problemSetInputs),
				)
			} else {
				// Check if a problem set already exists for this session
				existingPS, err := s.repo.GetProblemSetBySession(ctx, sessionID, ownerSub)
				if err != nil && !errors.Is(err, sharedRepo.ErrNotFound) {
					logger.Error("failed to check for existing problem set",
						"session_id", sessionID,
						"error", err.Error(),
					)
				} else if existingPS != nil {
					logger.Info("problem set already exists for this session, skipping creation",
						"session_id", sessionID,
						"problem_set_id", existingPS.ID,
					)
				} else {
					// No existing problem set, create one
					problemSetTasks, err := ConvertToProblemSetTasks(problemSetInputs)
					if err != nil {
						logger.Error("failed to convert problem set tasks",
							"session_id", sessionID,
							"error", err.Error(),
						)
					} else {
						newProblemSet := domain.ProblemSet{
							TutorialID:            sess.TutorialID,
							OwnerSub:              ownerSub,
							WeekOf:                weekOf,
							AssignedFromSessionID: sessionID,
							Status:                "assigned",
							Tasks:                 problemSetTasks,
						}
						_, err = s.repo.CreateProblemSet(ctx, ownerSub, newProblemSet)
						if err != nil {
							logger.Error("failed to create problem set",
								"session_id", sessionID,
								"error", err.Error(),
							)
						} else {
							logger.Info("successfully created problem set from agent response",
								"session_id", sessionID,
								"task_count", len(problemSetTasks),
							)
						}
					}
				}
			}
		}
	}

	// Strip the diagnostic and problem set JSON blocks from the agent response before storing
	agentTextStripped := StripDiagnosticBlock(agentText)
	agentTextStripped = StripProblemSetBlock(agentTextStripped)

	// If the stripped text is empty, delete the agent turn, mark user turn as failed, and return
	if strings.TrimSpace(agentTextStripped) == "" {
		logger.Warn("agent response is empty after stripping blocks",
			"session_id", sessionID,
			"turn_id", agentCreated.ID,
		)
		// Delete the empty agent turn
		if delErr := s.repo.DeleteTutorialTurn(ctx, agentCreated.ID, sessionID, ownerSub); delErr != nil {
			logger.Error("failed to delete empty agent turn",
				"session_id", sessionID,
				"turn_id", agentCreated.ID,
				"error", delErr.Error(),
			)
		}
		// Mark the user turn as failed
		if _, markErr := s.repo.MarkTutorialTurnFailed(ctx, created.ID, sessionID, ownerSub); markErr != nil {
			logger.Error("failed to mark user turn as failed",
				"session_id", sessionID,
				"turn_id", created.ID,
				"error", markErr.Error(),
			)
		}
		s.hub.PublishError(sessionID, "Agent response was empty")
		return result, nil
	}

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

// ── Command parsing ────────────────────────────────────────────────────────────

// tutorialCommand identifies a recognized slash command in a tutorial turn.
type tutorialCommand int

const (
	tutorialCommandNone       tutorialCommand = iota
	tutorialCommandProblemSet                 // /problem-set
)

// problemSetCommandOptions holds parsed options for the /problem-set command.
type problemSetCommandOptions struct {
	Patterns   string // "auto" or comma-separated pattern codes
	Difficulty string // "beginner", "intermediate", or "advanced"
	Mode       string // "preview" or "commit"
}

// parseAndValidateTutorialCommand parses a slash command from text and validates
// it against the current session state. Returns tutorialCommandNone if text is
// not a slash command, or a ValidationError if the command is rejected.
func parseAndValidateTutorialCommand(text string, sess *domain.TutorialSession) (tutorialCommand, error) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return tutorialCommandNone, nil
	}

	// Extract the command token (first whitespace-delimited word).
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		// A bare "/" with nothing following it is not a valid command.
		return tutorialCommandNone, &ValidationError{
			Field:   "text",
			Message: "invalid command: missing command name after /",
		}
	}
	token := fields[0]
	switch token {
	case "/problem-set":
		if sess.Kind != domain.TutorialSessionKindExtended {
			return tutorialCommandNone, &ValidationError{
				Field:   "text",
				Message: "/problem-set is only available in extended tutorial sessions",
			}
		}
		return tutorialCommandProblemSet, nil
	default:
		return tutorialCommandNone, &ValidationError{
			Field:   "text",
			Message: fmt.Sprintf("unknown command: %s", token),
		}
	}
}

// parseProblemSetCommandOptions parses options from a /problem-set command.
// Returns default values for any unspecified options.
func parseProblemSetCommandOptions(text string) (problemSetCommandOptions, error) {
	// Default values
	opts := problemSetCommandOptions{
		Patterns:   "auto",
		Difficulty: "intermediate",
		Mode:       "commit",
	}

	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || fields[0] != "/problem-set" {
		return opts, fmt.Errorf("text is not a /problem-set command")
	}

	// Parse options starting after the command token
	for i := 1; i < len(fields); i++ {
		opt := fields[i]
		if !strings.HasPrefix(opt, "/") {
			continue // Skip non-option tokens
		}

		optName := opt[1:] // Remove leading /
		switch optName {
		case "patterns":
			if i+1 >= len(fields) {
				return opts, &ValidationError{
					Field:   "text",
					Message: "/patterns option requires a value (e.g., /patterns auto or /patterns TEXT_DRIFT,UNDEFINED_TERMS)",
				}
			}
			i++
			opts.Patterns = fields[i]

		case "difficulty":
			if i+1 >= len(fields) {
				return opts, &ValidationError{
					Field:   "text",
					Message: "/difficulty option requires a value: beginner, intermediate, or advanced",
				}
			}
			i++
			difficulty := fields[i]
			if difficulty != "beginner" && difficulty != "intermediate" && difficulty != "advanced" {
				return opts, &ValidationError{
					Field: "text",
					Message: fmt.Sprintf(
						"invalid difficulty: %s (must be beginner, intermediate, or advanced)",
						difficulty,
					),
				}
			}
			opts.Difficulty = difficulty

		case "mode":
			if i+1 >= len(fields) {
				return opts, &ValidationError{
					Field:   "text",
					Message: "/mode option requires a value: preview or commit",
				}
			}
			i++
			mode := fields[i]
			if mode != "preview" && mode != "commit" {
				return opts, &ValidationError{
					Field:   "text",
					Message: fmt.Sprintf("invalid mode: %s (must be preview or commit)", mode),
				}
			}
			opts.Mode = mode

		default:
			return opts, &ValidationError{
				Field:   "text",
				Message: fmt.Sprintf("unknown /problem-set option: /%s", optName),
			}
		}
	}

	return opts, nil
}

// mapCommandDifficultyToPromptDifficulty maps user-facing difficulty levels to prompt difficulty levels.
// beginner -> basic, intermediate -> standard, advanced -> rigorous
func mapCommandDifficultyToPromptDifficulty(cmdDifficulty string) string {
	switch cmdDifficulty {
	case "beginner":
		return "basic"
	case "intermediate":
		return "standard"
	case "advanced":
		return "rigorous"
	default:
		return "standard" // fallback to standard
	}
}

// sundayOfWeek returns the date of the Sunday that begins the ISO week containing t (UTC).
// If t is already a Sunday, it returns t's date at midnight UTC.
func sundayOfWeek(t time.Time) time.Time {
	t = t.UTC()
	daysFromSunday := int(t.Weekday()) // time.Sunday == 0
	sunday := t.AddDate(0, 0, -daysFromSunday)
	return time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 0, 0, 0, 0, time.UTC)
}
