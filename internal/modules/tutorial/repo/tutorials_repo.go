package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
)

// TutorialRepo handles all database operations for tutorials, tutorial sessions,
// and artifacts. Every method that accesses user-owned rows accepts ownerSub and
// enforces it in the WHERE clause, making cross-owner reads structurally impossible.
type TutorialRepo struct {
	repo.Base
}

// NewTutorialRepo constructs a TutorialRepo backed by the shared connection pool.
func NewTutorialRepo(b repo.Base) *TutorialRepo {
	return &TutorialRepo{Base: b}
}

// ── Tutorials ─────────────────────────────────────────────────────────────────

// CreateTutorial inserts a new tutorial and returns the fully-populated record.
func (r *TutorialRepo) CreateTutorial(
	ctx context.Context,
	ownerSub string,
	t domain.Tutorial,
) (*domain.Tutorial, error) {
	const q = `
		INSERT INTO tutorials
			(owner_sub, title, subject, description, difficulty)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, owner_sub, title, subject,
		          COALESCE(description,''), difficulty,
		          created_at, updated_at`

	row := r.Pool.QueryRow(ctx, q,
		ownerSub, t.Title, t.Subject, nvlTutStr(t.Description), t.Difficulty)
	return scanTutorial(row)
}

// GetTutorialByID returns the tutorial with the given id, enforcing owner_sub.
// Returns repo.ErrNotFound if no matching row exists.
func (r *TutorialRepo) GetTutorialByID(ctx context.Context, id, ownerSub string) (*domain.Tutorial, error) {
	const q = `
		SELECT id, owner_sub, title, subject,
		       COALESCE(description,''), difficulty,
		       created_at, updated_at
		FROM tutorials
		WHERE id = $1 AND owner_sub = $2`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	return scanTutorial(row)
}

// ListTutorials returns all tutorials owned by ownerSub, newest first.
func (r *TutorialRepo) ListTutorials(ctx context.Context, ownerSub string) ([]domain.Tutorial, error) {
	const q = `
		SELECT id, owner_sub, title, subject,
		       COALESCE(description,''), difficulty,
		       created_at, updated_at
		FROM tutorials
		WHERE owner_sub = $1
		ORDER BY created_at DESC`

	rows, err := r.Pool.Query(ctx, q, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list tutorials query: %w", err)
	}
	defer rows.Close()

	var result []domain.Tutorial
	for rows.Next() {
		t, err := scanTutorial(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *t)
	}
	return result, rows.Err()
}

// UpdateTutorial applies a partial patch to the tutorial and returns the updated record.
// Returns repo.ErrNotFound if the tutorial does not exist or belongs to a different owner.
func (r *TutorialRepo) UpdateTutorial(
	ctx context.Context,
	id, ownerSub string,
	patch domain.TutorialPatch,
) (*domain.Tutorial, error) {
	const q = `
		UPDATE tutorials
		SET title       = COALESCE($3, title),
		    subject     = COALESCE($4, subject),
		    description = COALESCE($5, description),
		    difficulty  = COALESCE($6, difficulty),
		    updated_at  = now()
		WHERE id = $1 AND owner_sub = $2
		RETURNING id, owner_sub, title, subject,
		          COALESCE(description,''), difficulty,
		          created_at, updated_at`

	row := r.Pool.QueryRow(ctx, q,
		id, ownerSub,
		patch.Title, patch.Subject, patch.Description, patch.Difficulty)
	return scanTutorial(row)
}

// DeleteTutorial removes the tutorial. Returns repo.ErrNotFound if it does not
// exist or the caller does not own it.
func (r *TutorialRepo) DeleteTutorial(ctx context.Context, id, ownerSub string) error {
	const q = `DELETE FROM tutorials WHERE id = $1 AND owner_sub = $2`
	tag, err := r.Pool.Exec(ctx, q, id, ownerSub)
	if err != nil {
		return fmt.Errorf("delete tutorial: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// ── TutorialSessions ──────────────────────────────────────────────────────────

// CreateSession inserts a new tutorial session and returns the fully-populated record.
func (r *TutorialRepo) CreateSession(
	ctx context.Context,
	ownerSub string,
	s domain.TutorialSession,
) (*domain.TutorialSession, error) {
	const q = `
		INSERT INTO tutorial_sessions (tutorial_id, owner_sub, kind)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id, tutorial_id, owner_sub, status, kind,
		          COALESCE(notes,''), started_at, ended_at`

	row := r.Pool.QueryRow(ctx, q, s.TutorialID, ownerSub, s.Kind)
	return scanTutorialSession(row)
}

// GetSessionByID returns the tutorial session with the given id, enforcing owner_sub.
// Returns repo.ErrNotFound if no matching row exists.
func (r *TutorialRepo) GetSessionByID(ctx context.Context, id, ownerSub string) (*domain.TutorialSession, error) {
	const q = `
		SELECT id, tutorial_id, owner_sub, status, kind,
		       COALESCE(notes,''), started_at, ended_at
		FROM tutorial_sessions
		WHERE id = $1 AND owner_sub = $2`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	return scanTutorialSession(row)
}

// ListSessionsByTutorialID returns all sessions for a tutorial in
// reverse-chronological order. Ownership is enforced via the owner_sub column.
func (r *TutorialRepo) ListSessionsByTutorialID(
	ctx context.Context,
	tutorialID, ownerSub string,
) ([]domain.TutorialSession, error) {
	const q = `
		SELECT id, tutorial_id, owner_sub, status, kind,
		       COALESCE(notes,''), started_at, ended_at
		FROM tutorial_sessions
		WHERE tutorial_id = $1 AND owner_sub = $2
		ORDER BY started_at DESC`

	rows, err := r.Pool.Query(ctx, q, tutorialID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list tutorial sessions by tutorial: %w", err)
	}
	defer rows.Close()

	var result []domain.TutorialSession
	for rows.Next() {
		s, err := scanTutorialSession(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tutorial sessions iterate: %w", err)
	}
	if result == nil {
		result = []domain.TutorialSession{}
	}
	return result, nil
}

// CompleteSession marks a session as complete and records ended_at.
// Returns repo.ErrNotFound when no matching in-progress session is found.
func (r *TutorialRepo) CompleteSession(
	ctx context.Context,
	id, ownerSub, notes string,
) (*domain.TutorialSession, error) {
	const q = `
		UPDATE tutorial_sessions
		SET status   = 'complete',
		    notes    = COALESCE(NULLIF($3, ''), notes),
		    ended_at = now()
		WHERE id = $1 AND owner_sub = $2
		  AND status = 'in_progress'
		RETURNING id, tutorial_id, owner_sub, status, kind,
		          COALESCE(notes,''), started_at, ended_at`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub, notes)
	sess, err := scanTutorialSession(row)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	return sess, nil
}

// AbandonSession marks a session as abandoned and records ended_at.
// Returns repo.ErrNotFound when no matching in-progress session is found.
func (r *TutorialRepo) AbandonSession(ctx context.Context, id, ownerSub string) (*domain.TutorialSession, error) {
	const q = `
		UPDATE tutorial_sessions
		SET status   = 'abandoned',
		    ended_at = now()
		WHERE id = $1 AND owner_sub = $2
		  AND status = 'in_progress'
		RETURNING id, tutorial_id, owner_sub, status, kind,
		          COALESCE(notes,''), started_at, ended_at`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	sess, err := scanTutorialSession(row)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	return sess, nil
}

// DeleteSession removes a tutorial session and all its associated artifacts.
// Returns repo.ErrNotFound if the session does not exist or belongs to another owner.
func (r *TutorialRepo) DeleteSession(ctx context.Context, id, ownerSub string) error {
	const q = `DELETE FROM tutorial_sessions WHERE id = $1 AND owner_sub = $2`
	tag, err := r.Pool.Exec(ctx, q, id, ownerSub)
	if err != nil {
		return fmt.Errorf("delete tutorial session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// ── Artifacts ─────────────────────────────────────────────────────────────────

// CreateArtifact inserts a new artifact and returns the fully-populated record.
func (r *TutorialRepo) CreateArtifact(
	ctx context.Context,
	ownerSub string,
	a domain.Artifact,
) (*domain.Artifact, error) {
	const q = `
		INSERT INTO artifacts (session_id, owner_sub, kind, title, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, owner_sub, kind, title, content, created_at`

	row := r.Pool.QueryRow(ctx, q, a.SessionID, ownerSub, string(a.Kind), a.Title, a.Content)
	return scanArtifact(row)
}

// GetArtifactByID returns the artifact with the given id, enforcing owner_sub.
// Returns repo.ErrNotFound if no matching row exists.
func (r *TutorialRepo) GetArtifactByID(ctx context.Context, id, ownerSub string) (*domain.Artifact, error) {
	const q = `
		SELECT id, session_id, owner_sub, kind, title, content, created_at
		FROM artifacts
		WHERE id = $1 AND owner_sub = $2`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub)
	return scanArtifact(row)
}

// ListArtifactsBySessionID returns all artifacts for a session in
// chronological order. Ownership is enforced via the owner_sub column.
func (r *TutorialRepo) ListArtifactsBySessionID(
	ctx context.Context,
	sessionID, ownerSub string,
) ([]domain.Artifact, error) {
	const q = `
		SELECT id, session_id, owner_sub, kind, title, content, created_at
		FROM artifacts
		WHERE session_id = $1 AND owner_sub = $2
		ORDER BY created_at`

	rows, err := r.Pool.Query(ctx, q, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list artifacts by session: %w", err)
	}
	defer rows.Close()

	var result []domain.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list artifacts iterate: %w", err)
	}
	if result == nil {
		result = []domain.Artifact{}
	}
	return result, nil
}

// DeleteArtifact removes the artifact.
// Returns repo.ErrNotFound if it does not exist or the caller does not own it.
func (r *TutorialRepo) DeleteArtifact(ctx context.Context, id, ownerSub string) error {
	const q = `DELETE FROM artifacts WHERE id = $1 AND owner_sub = $2`
	tag, err := r.Pool.Exec(ctx, q, id, ownerSub)
	if err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// ── Tutorial Turns ────────────────────────────────────────────────────────────

// CreateTutorialTurn inserts a new turn for the given tutorial session.
func (r *TutorialRepo) CreateTutorialTurn(
	ctx context.Context,
	sessionID, ownerSub string,
	turn domain.TutorialTurn,
) (*domain.TutorialTurn, error) {
	// First verify the session exists and the owner matches.
	const checkQ = `SELECT 1 FROM tutorial_sessions WHERE id = $1 AND owner_sub = $2`
	var exists int
	if err := r.Pool.QueryRow(ctx, checkQ, sessionID, ownerSub).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("check tutorial session ownership: %w", err)
	}

	const q = `
		INSERT INTO tutorial_turns (session_id, speaker, text)
		VALUES ($1, $2, $3)
		RETURNING id, session_id, speaker, text, created_at`

	row := r.Pool.QueryRow(ctx, q, sessionID, turn.Speaker, turn.Text)
	return scanTutorialTurn(row)
}

// ListTutorialTurns returns all turns for a tutorial session, ordered by creation time.
func (r *TutorialRepo) ListTutorialTurns(
	ctx context.Context,
	sessionID, ownerSub string,
) ([]domain.TutorialTurn, error) {
	// First verify the session exists and the owner matches.
	const checkQ = `SELECT 1 FROM tutorial_sessions WHERE id = $1 AND owner_sub = $2`
	var exists int
	if err := r.Pool.QueryRow(ctx, checkQ, sessionID, ownerSub).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("check tutorial session ownership: %w", err)
	}

	const q = `
		SELECT id, session_id, speaker, text, created_at
		FROM tutorial_turns
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.Pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list tutorial turns query: %w", err)
	}
	defer rows.Close()

	var result []domain.TutorialTurn
	for rows.Next() {
		t, err := scanTutorialTurn(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tutorial turns iterate: %w", err)
	}
	if result == nil {
		result = []domain.TutorialTurn{}
	}
	return result, nil
}

// UpdateTutorialTurn updates the text field of an existing turn.
// This is used when streaming agent responses to update the turn after
// it has been created with empty/placeholder text.
func (r *TutorialRepo) UpdateTutorialTurn(
	ctx context.Context,
	turnID, sessionID, ownerSub, newText string,
) (*domain.TutorialTurn, error) {
	// First verify the session exists and the owner matches.
	const checkQ = `SELECT 1 FROM tutorial_sessions WHERE id = $1 AND owner_sub = $2`
	var exists int
	if err := r.Pool.QueryRow(ctx, checkQ, sessionID, ownerSub).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("check tutorial session ownership: %w", err)
	}

	const q = `
		UPDATE tutorial_turns
		SET text = $1
		WHERE id = $2 AND session_id = $3
		RETURNING id, session_id, speaker, text, created_at`

	row := r.Pool.QueryRow(ctx, q, newText, turnID, sessionID)
	return scanTutorialTurn(row)
}

// ── Diagnostic Entries ────────────────────────────────────────────────────────

// CreateDiagnosticEntry inserts a new diagnostic entry and returns the fully-populated record.
func (r *TutorialRepo) CreateDiagnosticEntry(
	ctx context.Context,
	ownerSub string,
	entry domain.DiagnosticEntry,
) (*domain.DiagnosticEntry, error) {
	const q = `
		INSERT INTO diagnostic_entries
			(tutorial_id, tutorial_session_id, owner_sub, week_of, 
			 pattern_code, severity, status, evidence, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tutorial_id, tutorial_session_id, owner_sub, week_of,
		          pattern_code, severity, status, evidence, 
		          COALESCE(notes, ''), created_at, updated_at`

	row := r.Pool.QueryRow(ctx, q,
		entry.TutorialID, entry.TutorialSessionID, ownerSub, entry.WeekOf,
		entry.PatternCode, entry.Severity, entry.Status, entry.Evidence,
		nvlTutStr(entry.Notes))
	return scanDiagnosticEntry(row)
}

// ListDiagnosticEntriesByTutorial returns all diagnostic entries for a tutorial.
func (r *TutorialRepo) ListDiagnosticEntriesByTutorial(
	ctx context.Context,
	tutorialID, ownerSub string,
) ([]domain.DiagnosticEntry, error) {
	const q = `
		SELECT id, tutorial_id, tutorial_session_id, owner_sub, week_of,
		       pattern_code, severity, status, evidence,
		       COALESCE(notes, ''), created_at, updated_at
		FROM diagnostic_entries
		WHERE tutorial_id = $1 AND owner_sub = $2
		ORDER BY week_of DESC, created_at DESC`

	rows, err := r.Pool.Query(ctx, q, tutorialID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list diagnostic entries query: %w", err)
	}
	defer rows.Close()

	var result []domain.DiagnosticEntry
	for rows.Next() {
		e, err := scanDiagnosticEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	if result == nil {
		result = []domain.DiagnosticEntry{}
	}
	return result, rows.Err()
}

// ListDiagnosticEntriesByWeek returns diagnostic entries for a specific week.
func (r *TutorialRepo) ListDiagnosticEntriesByWeek(
	ctx context.Context,
	tutorialID, ownerSub string,
	weekOf string,
) ([]domain.DiagnosticEntry, error) {
	const q = `
		SELECT id, tutorial_id, tutorial_session_id, owner_sub, week_of,
		       pattern_code, severity, status, evidence,
		       COALESCE(notes, ''), created_at, updated_at
		FROM diagnostic_entries
		WHERE tutorial_id = $1 AND owner_sub = $2 AND week_of = $3
		ORDER BY created_at DESC`

	rows, err := r.Pool.Query(ctx, q, tutorialID, ownerSub, weekOf)
	if err != nil {
		return nil, fmt.Errorf("list diagnostic entries by week query: %w", err)
	}
	defer rows.Close()

	var result []domain.DiagnosticEntry
	for rows.Next() {
		e, err := scanDiagnosticEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	if result == nil {
		result = []domain.DiagnosticEntry{}
	}
	return result, rows.Err()
}

// ListRecentDiagnosticEntries returns the N most recent diagnostic entries for a tutorial.
func (r *TutorialRepo) ListRecentDiagnosticEntries(
	ctx context.Context,
	tutorialID, ownerSub string,
	limit int,
) ([]domain.DiagnosticEntry, error) {
	const q = `
		SELECT id, tutorial_id, tutorial_session_id, owner_sub, week_of,
		       pattern_code, severity, status, evidence,
		       COALESCE(notes, ''), created_at, updated_at
		FROM diagnostic_entries
		WHERE tutorial_id = $1 AND owner_sub = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := r.Pool.Query(ctx, q, tutorialID, ownerSub, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent diagnostic entries query: %w", err)
	}
	defer rows.Close()

	var result []domain.DiagnosticEntry
	for rows.Next() {
		e, err := scanDiagnosticEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *e)
	}
	if result == nil {
		result = []domain.DiagnosticEntry{}
	}
	return result, rows.Err()
}

// UpdateDiagnosticEntryStatus updates the status field of a diagnostic entry.
func (r *TutorialRepo) UpdateDiagnosticEntryStatus(
	ctx context.Context,
	id, ownerSub string,
	status domain.DiagnosticStatus,
) (*domain.DiagnosticEntry, error) {
	const q = `
		UPDATE diagnostic_entries
		SET status = $3, updated_at = now()
		WHERE id = $1 AND owner_sub = $2
		RETURNING id, tutorial_id, tutorial_session_id, owner_sub, week_of,
		          pattern_code, severity, status, evidence,
		          COALESCE(notes, ''), created_at, updated_at`

	row := r.Pool.QueryRow(ctx, q, id, ownerSub, status)
	return scanDiagnosticEntry(row)
}

// ── Problem Sets ──────────────────────────────────────────────────────────────

// CreateProblemSet inserts a new problem set and returns the fully-populated record.
func (r *TutorialRepo) CreateProblemSet(
	ctx context.Context,
	ownerSub string,
	ps domain.ProblemSet,
) (*domain.ProblemSet, error) {
	const q = `
		INSERT INTO problem_sets
			(tutorial_id, owner_sub, week_of, assigned_from_session_id, 
			 status, tasks, review_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tutorial_id, owner_sub, week_of, 
		          COALESCE(assigned_from_session_id, ''), status, tasks,
		          COALESCE(review_notes, ''), created_at, updated_at`

	row := r.Pool.QueryRow(ctx, q,
		ps.TutorialID, ownerSub, ps.WeekOf, nvlTutStr(ps.AssignedFromSessionID),
		ps.Status, ps.Tasks, nvlTutStr(ps.ReviewNotes))
	return scanProblemSet(row)
}

// GetProblemSetByWeek returns the problem set for a specific tutorial and week.
func (r *TutorialRepo) GetProblemSetByWeek(
	ctx context.Context,
	tutorialID, ownerSub, weekOf string,
) (*domain.ProblemSet, error) {
	const q = `
		SELECT id, tutorial_id, owner_sub, week_of,
		       COALESCE(assigned_from_session_id, ''), status, tasks,
		       COALESCE(review_notes, ''), created_at, updated_at
		FROM problem_sets
		WHERE tutorial_id = $1 AND owner_sub = $2 AND week_of = $3`

	row := r.Pool.QueryRow(ctx, q, tutorialID, ownerSub, weekOf)
	return scanProblemSet(row)
}

// ListProblemSets returns all problem sets for a tutorial.
func (r *TutorialRepo) ListProblemSets(
	ctx context.Context,
	tutorialID, ownerSub string,
) ([]domain.ProblemSet, error) {
	const q = `
		SELECT id, tutorial_id, owner_sub, week_of,
		       COALESCE(assigned_from_session_id, ''), status, tasks,
		       COALESCE(review_notes, ''), created_at, updated_at
		FROM problem_sets
		WHERE tutorial_id = $1 AND owner_sub = $2
		ORDER BY week_of DESC`

	rows, err := r.Pool.Query(ctx, q, tutorialID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list problem sets query: %w", err)
	}
	defer rows.Close()

	var result []domain.ProblemSet
	for rows.Next() {
		ps, err := scanProblemSet(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *ps)
	}
	if result == nil {
		result = []domain.ProblemSet{}
	}
	return result, rows.Err()
}

// LinkProblemSetPattern creates a link between a problem set and a diagnostic entry.
func (r *TutorialRepo) LinkProblemSetPattern(
	ctx context.Context,
	problemSetID, diagnosticEntryID, patternCode string,
) error {
	const q = `
		INSERT INTO problem_set_pattern_links
			(problem_set_id, diagnostic_entry_id, pattern_code)
		VALUES ($1, $2, $3)
		ON CONFLICT (problem_set_id, diagnostic_entry_id) DO NOTHING`

	_, err := r.Pool.Exec(ctx, q, problemSetID, diagnosticEntryID, patternCode)
	if err != nil {
		return fmt.Errorf("link problem set pattern: %w", err)
	}
	return nil
}

// ListPatternLinksForProblemSet returns all diagnostic entry IDs linked to a problem set.
func (r *TutorialRepo) ListPatternLinksForProblemSet(
	ctx context.Context,
	problemSetID string,
) ([]domain.ProblemSetPatternLink, error) {
	const q = `
		SELECT problem_set_id, diagnostic_entry_id, pattern_code
		FROM problem_set_pattern_links
		WHERE problem_set_id = $1`

	rows, err := r.Pool.Query(ctx, q, problemSetID)
	if err != nil {
		return nil, fmt.Errorf("list pattern links query: %w", err)
	}
	defer rows.Close()

	var result []domain.ProblemSetPatternLink
	for rows.Next() {
		var link domain.ProblemSetPatternLink
		if err := rows.Scan(&link.ProblemSetID, &link.DiagnosticEntryID, &link.PatternCode); err != nil {
			return nil, fmt.Errorf("scan pattern link: %w", err)
		}
		result = append(result, link)
	}
	if result == nil {
		result = []domain.ProblemSetPatternLink{}
	}
	return result, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

type tutorialScanner interface {
	Scan(dest ...any) error
}

func scanTutorial(row tutorialScanner) (*domain.Tutorial, error) {
	var t domain.Tutorial
	err := row.Scan(
		&t.ID, &t.OwnerSub,
		&t.Title, &t.Subject, &t.Description, &t.Difficulty,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan tutorial: %w", err)
	}
	return &t, nil
}

type tutorialSessionScanner interface {
	Scan(dest ...any) error
}

func scanTutorialSession(row tutorialSessionScanner) (*domain.TutorialSession, error) {
	var s domain.TutorialSession
	var status string
	var kind *string
	err := row.Scan(
		&s.ID, &s.TutorialID, &s.OwnerSub,
		&status, &kind, &s.Notes,
		&s.StartedAt, &s.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan tutorial session: %w", err)
	}
	s.Status = domain.TutorialSessionStatus(status)
	if kind != nil {
		s.Kind = domain.TutorialSessionKind(*kind)
	}
	return &s, nil
}

type artifactScanner interface {
	Scan(dest ...any) error
}

func scanArtifact(row artifactScanner) (*domain.Artifact, error) {
	var a domain.Artifact
	var kind string
	err := row.Scan(
		&a.ID, &a.SessionID, &a.OwnerSub,
		&kind, &a.Title, &a.Content,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan artifact: %w", err)
	}
	a.Kind = domain.ArtifactKind(kind)
	return &a, nil
}

type tutorialTurnScanner interface {
	Scan(dest ...any) error
}

func scanTutorialTurn(row tutorialTurnScanner) (*domain.TutorialTurn, error) {
	var t domain.TutorialTurn
	err := row.Scan(
		&t.ID, &t.SessionID, &t.Speaker, &t.Text, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan tutorial turn: %w", err)
	}
	return &t, nil
}

type diagnosticEntryScanner interface {
	Scan(dest ...any) error
}

func scanDiagnosticEntry(row diagnosticEntryScanner) (*domain.DiagnosticEntry, error) {
	var e domain.DiagnosticEntry
	var patternCode, status string
	err := row.Scan(
		&e.ID, &e.TutorialID, &e.TutorialSessionID, &e.OwnerSub, &e.WeekOf,
		&patternCode, &e.Severity, &status, &e.Evidence,
		&e.Notes, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan diagnostic entry: %w", err)
	}
	e.PatternCode = domain.DiagnosticPatternCode(patternCode)
	e.Status = domain.DiagnosticStatus(status)
	return &e, nil
}

type problemSetScanner interface {
	Scan(dest ...any) error
}

func scanProblemSet(row problemSetScanner) (*domain.ProblemSet, error) {
	var ps domain.ProblemSet
	err := row.Scan(
		&ps.ID, &ps.TutorialID, &ps.OwnerSub, &ps.WeekOf,
		&ps.AssignedFromSessionID, &ps.Status, &ps.Tasks,
		&ps.ReviewNotes, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, fmt.Errorf("scan problem set: %w", err)
	}
	return &ps, nil
}

// nvlTutStr converts an empty string to nil so pgx stores NULL for nullable columns.
func nvlTutStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
