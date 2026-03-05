package service

import (
	"context"
	"fmt"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
)

// phaseChangeMessages maps each target phase to a human-readable announcement
// that is stored as a system turn when the scheduler advances the phase.
var phaseChangeMessages = map[domain.SessionPhase]string{
	domain.PhaseOpposition:      "The reconstruction phase has ended. The session is now in the opposition phase.",
	domain.PhaseReversal:        "The opposition phase has ended. The session is now in the reversal phase.",
	domain.PhaseResidueRequired: "The reversal phase has ended. Please submit your residue statement to complete the session.",
	domain.PhaseDone:            "The session is complete.",
}

// InsertPhaseChangeTurn writes a system turn announcing that the session has
// advanced to toPhase. It is called by the scheduler immediately after a
// successful phase transition so the turn transcript reflects the change.
//
// Errors from this function are non-fatal with respect to the phase transition
// itself; callers should log but not roll back the transition on failure here.
func InsertPhaseChangeTurn(
	ctx context.Context,
	r *repo.SessionRepo,
	sessionID string,
	toPhase domain.SessionPhase,
) (*domain.Turn, error) {
	text, ok := phaseChangeMessages[toPhase]
	if !ok {
		text = fmt.Sprintf("Phase advanced to %s.", toPhase)
	}

	t := domain.Turn{
		SessionID: sessionID,
		Phase:     toPhase,
		Speaker:   "system",
		Text:      text,
		Flags:     []string{},
	}

	turn, err := r.InsertTurn(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("insert phase-change turn (session=%s phase=%s): %w", sessionID, toPhase, err)
	}
	return turn, nil
}
