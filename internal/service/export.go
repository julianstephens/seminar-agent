package service

import (
	"context"
	"fmt"

	"github.com/julianstephens/formation/internal/export"
	"github.com/julianstephens/formation/internal/repo"
)

// ExportService assembles full denormalized export payloads for seminars and
// sessions. It does not render them; rendering is handled by the export
// package renderers so that the service layer stays format-agnostic.
type ExportService struct {
	seminars *repo.SeminarRepo
	sessions *repo.SessionRepo
}

// NewExportService constructs an ExportService backed by the given repositories.
func NewExportService(seminars *repo.SeminarRepo, sessions *repo.SessionRepo) *ExportService {
	return &ExportService{seminars: seminars, sessions: sessions}
}

// ExportSeminar loads the seminar, its thesis history, and every session with
// turns, assembling them into a single SeminarExport.
// Returns NotFoundError when the seminar does not exist or is not owned by ownerSub.
func (s *ExportService) ExportSeminar(
	ctx context.Context,
	seminarID, ownerSub string,
) (*export.SeminarExport, error) {
	sem, err := s.seminars.GetByID(ctx, seminarID, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "seminar", seminarID)
	}

	sessions, err := s.sessions.ListBySeminarID(ctx, seminarID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("load sessions for export: %w", err)
	}

	sessionExports := make([]export.SessionExport, 0, len(sessions))
	for _, sess := range sessions {
		turns, err := s.sessions.ListTurns(ctx, sess.ID, ownerSub)
		if err != nil {
			return nil, fmt.Errorf("load turns for session %s: %w", sess.ID, err)
		}
		sessionExports = append(sessionExports, export.SessionExport{
			Session: sess,
			Turns:   turns,
		})
	}

	return &export.SeminarExport{
		Seminar:  *sem,
		Sessions: sessionExports,
	}, nil
}

// ExportSession loads the session and its turns, assembling them into a
// SessionExport.
// Returns NotFoundError when the session does not exist or is not owned by ownerSub.
func (s *ExportService) ExportSession(
	ctx context.Context,
	sessionID, ownerSub string,
) (*export.SessionExport, error) {
	sess, err := s.sessions.GetByID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "session", sessionID)
	}

	turns, err := s.sessions.ListTurns(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("load turns for export: %w", err)
	}

	return &export.SessionExport{
		Session: *sess,
		Turns:   turns,
	}, nil
}
