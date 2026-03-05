// Package repo_test contains integration tests for ownership enforcement.
//
// These tests require a live Postgres database and are skipped automatically
// when DATABASE_URL is not set in the environment. To run them locally:
//
//	DATABASE_URL=postgres://... go test ./internal/repo/... -run Integration -v
//
// The tests use a single shared pool and clean up their own rows via unique
// owner_sub prefixes, so they are safe to run against a development database
// that also carries real data.
package repo_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julianstephens/formation/internal/config"
	"github.com/julianstephens/formation/internal/db"
	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
)

// ── test harness ──────────────────────────────────────────────────────────────

// integrationPool returns a connection pool for integration tests, skipping
// the test if DATABASE_URL is not set.
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	cfg := &config.Config{DatabaseURL: dbURL}
	pool, err := db.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// uniqueSub returns a unique owner_sub string derived from the test name +
// timestamp so parallel tests and re-runs never share rows.
func uniqueSub(t *testing.T) string {
	t.Helper()
	return "test|" + t.Name() + "|" + time.Now().Format("20060102150405.999999999")
}

// seedSeminar creates a seminar owned by ownerSub and registers a cleanup that
// deletes it (and cascades to sessions/turns) when the test ends.
func seedSeminar(t *testing.T, sr *repo.SeminarRepo, ownerSub string) *domain.Seminar {
	t.Helper()

	sem, err := sr.Create(context.Background(), ownerSub, domain.Seminar{
		Title:               "Integration Test Seminar",
		ThesisCurrent:       "Integration thesis",
		DefaultMode:         "paperback",
		DefaultReconMinutes: 15,
	})
	if err != nil {
		t.Fatalf("seed seminar: %v", err)
	}
	t.Cleanup(func() {
		// Best-effort delete; ignore errors since the row may already be gone.
		_ = sr.Delete(context.Background(), sem.ID, ownerSub)
	})
	return sem
}

// ── Seminar ownership tests ───────────────────────────────────────────────────

// TestIntegration_SeminarRepo_GetByID_ownerEnforced verifies that GetByID
// returns ErrNotFound when the ownerSub does not match the row's owner.
func TestIntegration_SeminarRepo_GetByID_ownerEnforced(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	sr := repo.NewSeminarRepo(base)

	owner := uniqueSub(t)
	attacker := uniqueSub(t) + "_attacker"

	sem := seedSeminar(t, sr, owner)

	// Legitimate owner can read.
	_, err := sr.GetByID(context.Background(), sem.ID, owner)
	if err != nil {
		t.Fatalf("owner GetByID failed: %v", err)
	}

	// Different owner must get ErrNotFound, not the row.
	_, err = sr.GetByID(context.Background(), sem.ID, attacker)
	if err == nil {
		t.Fatal("expected ErrNotFound for cross-owner GetByID, got nil")
	}
}

// TestIntegration_SeminarRepo_List_isolatedByOwner verifies that List returns
// only the seminars owned by the requesting user.
func TestIntegration_SeminarRepo_List_isolatedByOwner(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	sr := repo.NewSeminarRepo(base)

	ownerA := uniqueSub(t) + "_A"
	ownerB := uniqueSub(t) + "_B"

	seedSeminar(t, sr, ownerA)
	seedSeminar(t, sr, ownerA)
	seedSeminar(t, sr, ownerB)

	listA, err := sr.List(context.Background(), ownerA)
	if err != nil {
		t.Fatalf("List(ownerA): %v", err)
	}
	for _, s := range listA {
		if s.OwnerSub == ownerB {
			t.Errorf("List(ownerA) returned seminar owned by ownerB: %+v", s)
		}
	}
	if len(listA) < 2 {
		t.Errorf("List(ownerA) returned %d results, expected ≥2", len(listA))
	}

	listB, err := sr.List(context.Background(), ownerB)
	if err != nil {
		t.Fatalf("List(ownerB): %v", err)
	}
	for _, s := range listB {
		if s.OwnerSub == ownerA {
			t.Errorf("List(ownerB) returned seminar owned by ownerA: %+v", s)
		}
	}
}

// TestIntegration_SeminarRepo_Delete_ownerEnforced verifies that Delete is a
// no-op when called with a different owner_sub (returns ErrNotFound, not an
// error that would expose the existence of the row).
func TestIntegration_SeminarRepo_Delete_ownerEnforced(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	sr := repo.NewSeminarRepo(base)

	owner := uniqueSub(t)
	attacker := uniqueSub(t) + "_attacker"

	sem := seedSeminar(t, sr, owner)

	// Cross-owner delete must fail.
	if err := sr.Delete(context.Background(), sem.ID, attacker); err == nil {
		t.Fatal("expected error for cross-owner Delete, got nil")
	}

	// Row must still exist for the real owner.
	_, err := sr.GetByID(context.Background(), sem.ID, owner)
	if err != nil {
		t.Fatalf("seminar should still exist after failed cross-owner delete: %v", err)
	}
}

// ── Session ownership tests ───────────────────────────────────────────────────

// seedSession creates a session for the given seminar and owner, with a
// future PhaseEndsAt so the scheduler would not immediately fire.
func seedSession(
	t *testing.T,
	sr *repo.SessionRepo,
	seminarID, ownerSub string,
) *domain.Session {
	t.Helper()

	now := time.Now().UTC()
	sess, err := sr.Create(context.Background(), ownerSub, domain.Session{
		SeminarID:      seminarID,
		SectionLabel:   "ch. 1",
		Mode:           "paperback",
		ReconMinutes:   15,
		PhaseStartedAt: now,
		PhaseEndsAt:    now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sess
}

// TestIntegration_SessionRepo_GetByID_ownerEnforced verifies that a session
// cannot be fetched by a different owner.
func TestIntegration_SessionRepo_GetByID_ownerEnforced(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	sr := repo.NewSeminarRepo(base)
	sessR := repo.NewSessionRepo(base)

	owner := uniqueSub(t)
	attacker := uniqueSub(t) + "_attacker"

	sem := seedSeminar(t, sr, owner)
	sess := seedSession(t, sessR, sem.ID, owner)

	// Legitimate owner can read.
	_, err := sessR.GetByID(context.Background(), sess.ID, owner)
	if err != nil {
		t.Fatalf("owner GetByID failed: %v", err)
	}

	// Different owner must get ErrNotFound.
	_, err = sessR.GetByID(context.Background(), sess.ID, attacker)
	if err == nil {
		t.Fatal("expected ErrNotFound for cross-owner session GetByID, got nil")
	}
}

// TestIntegration_SessionRepo_Abandon_ownerEnforced verifies that Abandon
// with a wrong owner does not affect the session.
func TestIntegration_SessionRepo_Abandon_ownerEnforced(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	sr := repo.NewSeminarRepo(base)
	sessR := repo.NewSessionRepo(base)

	owner := uniqueSub(t)
	attacker := uniqueSub(t) + "_attacker"

	sem := seedSeminar(t, sr, owner)
	sess := seedSession(t, sessR, sem.ID, owner)

	// Cross-owner abandon must fail.
	_, err := sessR.Abandon(context.Background(), sess.ID, attacker)
	if err == nil {
		t.Fatal("expected error for cross-owner Abandon, got nil")
	}

	// Session must still be in_progress.
	got, err := sessR.GetByID(context.Background(), sess.ID, owner)
	if err != nil {
		t.Fatalf("GetByID after failed abandon: %v", err)
	}
	if got.Status != domain.SessionStatusInProgress {
		t.Errorf("session status = %q after failed abandon, want in_progress", got.Status)
	}
}

// TestIntegration_SessionRepo_ListBySeminar_isolatedByOwner verifies that
// sessions cannot be listed across ownership boundaries.
func TestIntegration_SessionRepo_ListBySeminar_isolatedByOwner(t *testing.T) {
	pool := integrationPool(t)
	base := repo.Base{Pool: pool}
	srSeminar := repo.NewSeminarRepo(base)
	srSession := repo.NewSessionRepo(base)

	owner := uniqueSub(t)
	attacker := uniqueSub(t) + "_attacker"

	sem := seedSeminar(t, srSeminar, owner)
	seedSession(t, srSession, sem.ID, owner)

	// Attacker queries the same seminar_id but with their own sub.
	sessions, err := srSession.ListBySeminarID(context.Background(), sem.ID, attacker)
	if err != nil {
		t.Fatalf("ListBySeminarID(attacker): %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("attacker received %d sessions, want 0", len(sessions))
	}
}
