package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/julianstephens/formation/internal/agent"
	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/observability"
	"github.com/julianstephens/formation/internal/referee"
	"github.com/julianstephens/formation/internal/repo"
	"github.com/julianstephens/formation/internal/sse"
)

// ── TurnService ───────────────────────────────────────────────────────────────

// TurnService orchestrates user turn submission: guard checks, locator
// gating, prompt assembly, agent call, compliance rewrite, persistence, and
// SSE emission.
type TurnService struct {
	sessions  *repo.SessionRepo
	seminars  *repo.SeminarRepo
	assembler *agent.Assembler
	hub       *sse.Hub
	agent     agent.Provider // may be nil when no LLM is configured
}

// NewTurnService constructs a TurnService.
// provider may be nil; in that case the pipeline persists the user turn and
// assembles the prompt but skips the agent call, compliance check, and agent
// turn.
func NewTurnService(
	sessions *repo.SessionRepo,
	seminars *repo.SeminarRepo,
	assembler *agent.Assembler,
	hub *sse.Hub,
	provider agent.Provider,
) *TurnService {
	return &TurnService{
		sessions:  sessions,
		seminars:  seminars,
		assembler: assembler,
		hub:       hub,
		agent:     provider,
	}
}

// ── SubmitTurn ────────────────────────────────────────────────────────────────

// SubmitTurnParams holds the caller-supplied inputs for a user turn.
type SubmitTurnParams struct {
	Text string
}

// SubmitTurnResult carries the turns persisted by a successful submission.
type SubmitTurnResult struct {
	UserTurn  *domain.Turn
	AgentTurn *domain.Turn // nil when no agent client is configured
}

// SubmitTurn runs the full turn pipeline for a user submission:
//  1. Load session; assert turn is allowed (not terminal, phase permits turns,
//     timer not expired).
//  2. Referee check – locator gating for paperback mode.
//  3. Persist user turn with any policy flags.
//  4. Emit turn_added SSE for the user turn.
//  5. Load seminar metadata and prior turns; assemble the agent prompt.
//  6. Call agent (if configured).
//  7. Persist agent turn.
//  8. Emit turn_added SSE for the agent turn.
func (s *TurnService) SubmitTurn(
	ctx context.Context,
	sessionID, ownerSub string,
	p SubmitTurnParams,
) (*SubmitTurnResult, error) {
	if strings.TrimSpace(p.Text) == "" {
		return nil, &ValidationError{Field: "text", Message: "must not be blank"}
	}

	// 1. Load session and assert the turn is allowed.
	sess, err := s.sessions.GetByID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "session", sessionID)
	}
	if err := AssertTurnAllowed(sess); err != nil {
		return nil, err
	}

	// 2. Referee: locator gating.
	refResult, refErr := referee.Check(referee.Policy{Mode: sess.Mode}, p.Text)
	if refErr != nil {
		return nil, refErr
	}

	// 3. Persist user turn.
	userTurn, err := s.sessions.InsertTurn(ctx, domain.Turn{
		SessionID: sessionID,
		Phase:     sess.Phase,
		Speaker:   "user",
		Text:      p.Text,
		Flags:     refResult.Flags,
	})
	if err != nil {
		return nil, fmt.Errorf("persist user turn: %w", err)
	}

	// 4. Emit turn_added SSE for user turn.
	s.hub.PublishTurnAdded(userTurn)

	// 5. Load seminar and prior turns; assemble prompt.
	sem, err := s.seminars.GetByID(ctx, sess.SeminarID, ownerSub)
	if err != nil {
		return &SubmitTurnResult{UserTurn: userTurn},
			fmt.Errorf("load seminar for prompt assembly: %w", err)
	}

	priorTurns, err := s.sessions.ListTurns(ctx, sessionID, ownerSub)
	if err != nil {
		return &SubmitTurnResult{UserTurn: userTurn},
			fmt.Errorf("load prior turns for prompt assembly: %w", err)
	}

	messages, err := s.assembler.Assemble(agent.AssembleParams{
		SeminarTitle:  sem.Title,
		SeminarThesis: sem.ThesisCurrent,
		SectionLabel:  sess.SectionLabel,
		Mode:          sess.Mode,
		ExcerptText:   sess.ExcerptText,
		Phase:         sess.Phase,
		Turns:         priorTurns,
	})
	if err != nil {
		return &SubmitTurnResult{UserTurn: userTurn},
			fmt.Errorf("assemble agent prompt: %w", err)
	}
	_ = messages // consumed by agent call below; referenced here to keep assembly errors surfaced

	result := &SubmitTurnResult{UserTurn: userTurn}

	// 6-8. Call agent, run compliance check/rewrite, persist agent turn, emit SSE.
	if s.agent == nil {
		// No provider configured – return user turn only.
		return result, nil
	}

	// 6. Call the LLM.
	agentText, err := s.agent.Complete(ctx, messages)
	if err != nil {
		// Log the agent error with context for debugging
		logger := observability.LoggerFromContext(ctx)
		logger.Error("agent call failed",
			"session_id", sessionID,
			"error", err.Error(),
		)
		return result, fmt.Errorf("agent call: %w", err)
	}

	// 6b. Compliance check and optional rewrite.
	crResult, crErr := agent.ApplyCompliance(
		ctx,
		s.agent,
		s.assembler,
		agentText,
		string(sess.Phase),
		sess.Mode,
	)
	if crErr != nil {
		// Log but do not abort; crResult.Text is still usable.
		_ = fmt.Errorf("compliance rewrite (non-fatal): %w", crErr)
	}
	agentText = crResult.Text
	agentFlags := crResult.Flags // contains agent_rewrite when rewritten

	// 7. Persist agent turn with compliance flags.
	agentTurn, err := s.sessions.InsertTurn(ctx, domain.Turn{
		SessionID: sessionID,
		Phase:     sess.Phase,
		Speaker:   "agent",
		Text:      agentText,
		Flags:     agentFlags,
	})
	if err != nil {
		return result, fmt.Errorf("persist agent turn: %w", err)
	}

	// 8. Emit turn_added SSE for agent turn.
	s.hub.PublishTurnAdded(agentTurn)

	result.AgentTurn = agentTurn
	return result, nil
}
