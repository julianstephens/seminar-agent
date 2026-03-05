package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/julianstephens/formation/internal/domain"
)

// SessionRepo handles all database operations for sessions and turns.
// Every method that accesses user-owned rows accepts ownerSub and enforces it
// in the WHERE clause, making cross-owner reads structurally impossible.
type SessionRepo struct {
	Base
}

// NewSessionRepo constructs a SessionRepo backed by the shared connection pool.
func NewSessionRepo(b Base) *SessionRepo {
	return &SessionRepo{Base: b}
}

// ── Sessions ──────────────────────────────────────────────────────────────────

// Create inserts a new session and returns the fully-populated record.
func (r *SessionRepo) Create(ctx context.Context, ownerSub string, s domain.Session) (*domain.Session, error) {
	const q = `
		INSERT INTO sessions
			(seminar_id, owner_sub, section_label, mode,
			 excerpt_text, excerpt_hash, recon_minutes,
			 phase_started_at, phase_ends_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, seminar_id, owner_sub, section_label, mode,
		          COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		          status, phase, recon_minutes,
		          phase_started_at, phase_ends_at, started_at, ended_at,
		          COALESCE(residue_text,'')`

	row := r.Pool.QueryRow(ctx, q,
		s.SeminarID, ownerSub, s.SectionLabel, s.Mode,
		nvlStr(s.ExcerptText), nvlStr(s.ExcerptHash),
		s.ReconMinutes, s.PhaseStartedAt, s.PhaseEndsAt,
	)
	return scanSession(row)
}

// GetByID returns the session with the given id, enforcing owner_sub.
// Returns ErrNotFound if no matching row exists.
func (r *SessionRepo) GetByID(ctx context.Context, id, ownerSub string) (*domain.Session, error) {
	const q = `
		SELECT id, seminar_id, owner_sub, section_label, mode,
		       COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		       status, phase, recon_minutes,
		       phase_started_at, phase_ends_at, started_at, ended_at,
		       COALESCE(residue_text,'')
		FROM sessions
		WHERE id = $1 AND owner_sub = $2`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	return scanSession(row)
}

// Abandon marks an in_progress session as abandoned and records ended_at.
// Returns ErrNotFound if the session does not exist, belongs to another
// owner, or is already terminal (the UPDATE matches no rows).
func (r *SessionRepo) Abandon(ctx context.Context, id, ownerSub string) (*domain.Session, error) {
	const q = `
		UPDATE sessions
		SET status   = 'abandoned',
		    ended_at = now()
		WHERE id = $1 AND owner_sub = $2
		  AND status = 'in_progress'
		RETURNING id, seminar_id, owner_sub, section_label, mode,
		          COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		          status, phase, recon_minutes,
		          phase_started_at, phase_ends_at, started_at, ended_at,
		          COALESCE(residue_text,'')`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	sess, err := scanSession(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sess, nil
}

// SetResidue stores the residue text and advances the session to complete/done.
// The UPDATE is conditional on phase='residue_required' so it acts as a
// precondition check without a separate SELECT.
// Returns ErrNotFound if the session is not in residue_required phase,
// belongs to another owner, or does not exist.
func (r *SessionRepo) SetResidue(ctx context.Context, id, ownerSub, residueText string) (*domain.Session, error) {
	const q = `
		UPDATE sessions
		SET residue_text = $3,
		    phase        = 'done',
		    status       = 'complete',
		    ended_at     = now()
		WHERE id = $1 AND owner_sub = $2
		  AND phase  = 'residue_required'
		  AND status = 'in_progress'
		RETURNING id, seminar_id, owner_sub, section_label, mode,
		          COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		          status, phase, recon_minutes,
		          phase_started_at, phase_ends_at, started_at, ended_at,
		          COALESCE(residue_text,'')`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub, residueText)
	sess, err := scanSession(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sess, nil
}

// Delete removes a session and all associated turns.
// Returns ErrNotFound if the session does not exist or belongs to another owner.
func (r *SessionRepo) Delete(ctx context.Context, id, ownerSub string) error {
	// Delete associated turns first (cascade via foreign key would also work, but explicit is clearer)
	const deleteTurnsQ = `
		DELETE FROM turns
		WHERE session_id = $1
		  AND session_id IN (
			  SELECT id FROM sessions WHERE id = $1 AND owner_sub = $2
		  )`

	_, err := r.Pool.Exec(ctx, deleteTurnsQ, id, ownerSub)
	if err != nil {
		return fmt.Errorf("delete turns: %w", err)
	}

	// Delete the session itself
	const deleteSessionQ = `
		DELETE FROM sessions
		WHERE id = $1 AND owner_sub = $2`

	result, err := r.Pool.Exec(ctx, deleteSessionQ, id, ownerSub)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ── Turns ─────────────────────────────────────────────────────────────────────

// ListTurns returns all turns for a session in chronological order.
// Ownership is verified via a join to the sessions table.
func (r *SessionRepo) ListTurns(ctx context.Context, sessionID, ownerSub string) ([]domain.Turn, error) {
	const q = `
		SELECT t.id, t.session_id, t.phase, t.speaker, t.text, t.flags, t.created_at
		FROM turns t
		JOIN sessions s ON s.id = t.session_id
		WHERE t.session_id = $1 AND s.owner_sub = $2
		ORDER BY t.created_at`

	rows, err := r.Pool.Query(ctx, q, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list turns: %w", err)
	}
	defer rows.Close()

	var result []domain.Turn
	for rows.Next() {
		t, err := scanTurn(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list turns iterate: %w", err)
	}
	if result == nil {
		result = []domain.Turn{}
	}
	return result, nil
}

// ListBySeminarID returns all sessions for a seminar in reverse-chronological
// order. Ownership is enforced via the owner_sub column.
func (r *SessionRepo) ListBySeminarID(ctx context.Context, seminarID, ownerSub string) ([]domain.Session, error) {
	const q = `
		SELECT id, seminar_id, owner_sub, section_label, mode,
		       COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		       status, phase, recon_minutes,
		       phase_started_at, phase_ends_at, started_at, ended_at,
		       COALESCE(residue_text,'')
		FROM sessions
		WHERE seminar_id = $1 AND owner_sub = $2
		ORDER BY started_at DESC`

	rows, err := r.Pool.Query(ctx, q, seminarID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list sessions by seminar: %w", err)
	}
	defer rows.Close()

	var result []domain.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions by seminar iterate: %w", err)
	}
	if result == nil {
		result = []domain.Session{}
	}
	return result, nil
}

// ── Scheduler helpers ─────────────────────────────────────────────────────────

// ListInProgress returns all sessions that are in_progress and in a timed
// phase (reconstruction, opposition, or reversal). Used by the scheduler on
// startup to re-register phase timers.
func (r *SessionRepo) ListInProgress(ctx context.Context) ([]domain.Session, error) {
	const q = `
		SELECT id, seminar_id, owner_sub, section_label, mode,
		       COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		       status, phase, recon_minutes,
		       phase_started_at, phase_ends_at, started_at, ended_at,
		       COALESCE(residue_text,'')
		FROM sessions
		WHERE status = 'in_progress'
		  AND phase IN ('reconstruction', 'opposition', 'reversal')`

	rows, err := r.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list in-progress sessions: %w", err)
	}
	defer rows.Close()

	var result []domain.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list in-progress sessions iterate: %w", err)
	}
	return result, nil
}

// AdvancePhase atomically transitions sessionID from fromPhase to the next
// phase. It serializes concurrent callers with SELECT … FOR UPDATE so that
// only the first caller commits the transition; subsequent calls for the same
// (session, fromPhase) pair are race-safe no-ops.
//
// Returns (updatedSession, true, nil) when the transition was applied.
// Returns (nil, false, nil) when the session has already been advanced beyond
// fromPhase or is no longer in_progress (no-op, not an error).
// Returns (nil, false, ErrNotFound) when the session row does not exist.
func (r *SessionRepo) AdvancePhase(
	ctx context.Context,
	sessionID string,
	fromPhase domain.SessionPhase,
) (*domain.Session, bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("advance phase begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback on error is intentional

	// Lock the row to serialize concurrent scheduler callbacks.
	const lockQ = `
		SELECT id, phase, status, recon_minutes, phase_ends_at
		FROM sessions
		WHERE id = $1
		FOR UPDATE`

	var (
		rowID, rowPhase, rowStatus string
		reconMinutes               int
		currentPhaseEndsAt         time.Time
	)
	err = tx.QueryRow(ctx, lockQ, sessionID).Scan(
		&rowID, &rowPhase, &rowStatus, &reconMinutes, &currentPhaseEndsAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrNotFound
		}
		return nil, false, fmt.Errorf("advance phase lock row: %w", err)
	}

	// Race-safe no-op: session already advanced or is terminal.
	if domain.SessionPhase(rowPhase) != fromPhase ||
		domain.SessionStatus(rowStatus) != domain.SessionStatusInProgress {
		return nil, false, nil
	}

	next := domain.NextPhase(domain.SessionPhase(rowPhase))
	now := time.Now().UTC()

	// Compute new phase_ends_at for timed phases; keep existing value otherwise.
	newPhaseEndsAt := currentPhaseEndsAt
	switch next {
	case domain.PhaseOpposition, domain.PhaseReversal:
		newPhaseEndsAt = now.Add(time.Duration(reconMinutes) * time.Minute)
	}

	// Advancing to done also marks the session complete.
	newStatus := domain.SessionStatusInProgress
	var endedAt *time.Time
	if next == domain.PhaseDone {
		newStatus = domain.SessionStatusComplete
		endedAt = &now
	}

	const updateQ = `
		UPDATE sessions SET
			phase            = $2,
			status           = $3,
			phase_started_at = $4,
			phase_ends_at    = $5,
			ended_at         = $6
		WHERE id = $1
		RETURNING id, seminar_id, owner_sub, section_label, mode,
		          COALESCE(excerpt_text,''), COALESCE(excerpt_hash,''),
		          status, phase, recon_minutes,
		          phase_started_at, phase_ends_at, started_at, ended_at,
		          COALESCE(residue_text,'')`

	row := tx.QueryRow(ctx, updateQ,
		sessionID, string(next), string(newStatus), now, newPhaseEndsAt, endedAt,
	)
	sess, err := scanSession(row)
	if err != nil {
		return nil, false, fmt.Errorf("advance phase update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("advance phase commit: %w", err)
	}

	return sess, true, nil
}

// InsertTurn persists a single turn and returns the fully-populated record.
// Intended for use by the system-turn and agent-turn paths that do not require
// owner_sub enforcement (the session_id foreign key provides referential
// integrity).
func (r *SessionRepo) InsertTurn(ctx context.Context, t domain.Turn) (*domain.Turn, error) {
	flags, err := json.Marshal(t.Flags)
	if err != nil {
		return nil, fmt.Errorf("marshal turn flags: %w", err)
	}

	const q = `
		INSERT INTO turns (session_id, phase, speaker, text, flags)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, phase, speaker, text, flags, created_at`

	row := r.Pool.QueryRow(ctx, q,
		t.SessionID, string(t.Phase), t.Speaker, t.Text, flags,
	)
	return scanTurn(row)
}

// ── helpers ───────────────────────────────────────────────────────────────────

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSession(row sessionScanner) (*domain.Session, error) {
	var s domain.Session
	var status, phase string
	err := row.Scan(
		&s.ID, &s.SeminarID, &s.OwnerSub, &s.SectionLabel, &s.Mode,
		&s.ExcerptText, &s.ExcerptHash,
		&status, &phase,
		&s.ReconMinutes, &s.PhaseStartedAt, &s.PhaseEndsAt,
		&s.StartedAt, &s.EndedAt,
		&s.ResidueText,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	s.Status = domain.SessionStatus(status)
	s.Phase = domain.SessionPhase(phase)
	return &s, nil
}

type turnScanner interface {
	Scan(dest ...any) error
}

func scanTurn(row turnScanner) (*domain.Turn, error) {
	var t domain.Turn
	var phase string
	var rawFlags []byte
	if err := row.Scan(&t.ID, &t.SessionID, &phase, &t.Speaker, &t.Text, &rawFlags, &t.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan turn: %w", err)
	}
	t.Phase = domain.SessionPhase(phase)
	if rawFlags != nil {
		if err := json.Unmarshal(rawFlags, &t.Flags); err != nil {
			return nil, fmt.Errorf("unmarshal turn flags: %w", err)
		}
	}
	if t.Flags == nil {
		t.Flags = []string{}
	}
	return &t, nil
}

// nvlStr converts an empty string to nil so pgx stores NULL for nullable columns.
func nvlStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
