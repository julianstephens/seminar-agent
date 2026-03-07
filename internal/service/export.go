package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/julianstephens/formation/internal/export"
	seminarRepo "github.com/julianstephens/formation/internal/modules/seminar/repo"
	tutorialRepo "github.com/julianstephens/formation/internal/modules/tutorial/repo"
	"github.com/julianstephens/formation/internal/observability"
	"github.com/julianstephens/formation/internal/storage"
)

// ExportService assembles full denormalized export payloads for seminars and
// sessions. It does not render them; rendering is handled by the export
// package renderers so that the service layer stays format-agnostic.
type ExportService struct {
	seminars  *seminarRepo.SeminarRepo
	sessions  *seminarRepo.SessionRepo
	tutorials *tutorialRepo.TutorialRepo
	s3        *storage.S3Client
	log       *slog.Logger
}

// NewExportService constructs an ExportService backed by the given repositories.
func NewExportService(
	seminars *seminarRepo.SeminarRepo,
	sessions *seminarRepo.SessionRepo,
	tutorials *tutorialRepo.TutorialRepo,
) *ExportService {
	return &ExportService{seminars: seminars, sessions: sessions, tutorials: tutorials}
}

// WithS3 attaches an S3 client and logger to the service so that
// UploadAndPresign* methods are available.
func (s *ExportService) WithS3(client *storage.S3Client, logger *slog.Logger) *ExportService {
	s.s3 = client
	s.log = logger
	return s
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
		return nil, WrapNotFound(err, "seminar", seminarID)
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
		return nil, WrapNotFound(err, "session", sessionID)
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

// ExportTutorial loads the tutorial and every session with turns, assembling them
// into a single TutorialExport.
// Returns NotFoundError when the tutorial does not exist or is not owned by ownerSub.
func (s *ExportService) ExportTutorial(
	ctx context.Context,
	tutorialID, ownerSub string,
) (*export.TutorialExport, error) {
	tut, err := s.tutorials.GetTutorialByID(ctx, tutorialID, ownerSub)
	if err != nil {
		return nil, WrapNotFound(err, "tutorial", tutorialID)
	}

	sessions, err := s.tutorials.ListSessionsByTutorialID(ctx, tutorialID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("load sessions for tutorial export: %w", err)
	}

	sessionExports := make([]export.TutorialSessionExport, 0, len(sessions))
	for _, sess := range sessions {
		turns, err := s.tutorials.ListTutorialTurns(ctx, sess.ID, ownerSub)
		if err != nil {
			return nil, fmt.Errorf("load turns for tutorial session %s: %w", sess.ID, err)
		}
		artifacts, err := s.tutorials.ListArtifactsBySessionID(ctx, sess.ID, ownerSub)
		if err != nil {
			return nil, fmt.Errorf("load artifacts for tutorial session %s: %w", sess.ID, err)
		}
		sessionExports = append(sessionExports, export.TutorialSessionExport{
			Session:   sess,
			Turns:     turns,
			Artifacts: artifacts,
		})
	}

	return &export.TutorialExport{
		Tutorial: *tut,
		Sessions: sessionExports,
	}, nil
}

// ExportTutorialSession loads the tutorial session and its turns, assembling them
// into a TutorialSessionExport.
// Returns NotFoundError when the session does not exist or is not owned by ownerSub.
func (s *ExportService) ExportTutorialSession(
	ctx context.Context,
	sessionID, ownerSub string,
) (*export.TutorialSessionExport, error) {
	sess, err := s.tutorials.GetSessionByID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, WrapNotFound(err, "tutorial_session", sessionID)
	}

	turns, err := s.tutorials.ListTutorialTurns(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("load turns for tutorial session export: %w", err)
	}

	artifacts, err := s.tutorials.ListArtifactsBySessionID(ctx, sessionID, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("load artifacts for tutorial session export: %w", err)
	}

	return &export.TutorialSessionExport{
		Session:   *sess,
		Turns:     turns,
		Artifacts: artifacts,
	}, nil
}

// ── S3 upload + presign helpers ────────────────────────────────────────────────

// UploadAndPresignSeminar assembles the seminar export, renders it to the
// requested format, uploads it to S3, and returns a presigned download URL.
// Returns an error if the S3 upload fails.
func (s *ExportService) UploadAndPresignSeminar(
	ctx context.Context,
	seminarID, ownerSub, format string,
) (string, error) {
	logger := observability.LoggerFromContext(ctx)

	result, err := s.ExportSeminar(ctx, seminarID, ownerSub)
	if err != nil {
		return "", err
	}

	content, contentType, ext, err := renderExport(result, format,
		export.RenderSeminarMarkdown, export.RenderSeminarJSON)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("exports/seminars/%s.%s", seminarID, ext)
	if err := s.s3.Upload(ctx, key, content, contentType); err != nil {
		logger.Error("s3 upload failed", slog.String("key", key), slog.String("error", err.Error()))
		return "", fmt.Errorf("upload seminar export: %w", err)
	}

	url, err := s.s3.PresignURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("presign seminar export: %w", err)
	}

	logger.Info("seminar export uploaded",
		slog.String("seminar_id", seminarID),
		slog.String("format", format),
		slog.String("key", key),
	)
	return url, nil
}

// UploadAndPresignSession assembles the session export, renders it to the
// requested format, uploads it to S3, and returns a presigned download URL.
// Returns an error if the S3 upload fails.
func (s *ExportService) UploadAndPresignSession(
	ctx context.Context,
	sessionID, ownerSub, format string,
) (string, error) {
	logger := observability.LoggerFromContext(ctx)

	result, err := s.ExportSession(ctx, sessionID, ownerSub)
	if err != nil {
		return "", err
	}

	content, contentType, ext, err := renderExport(result, format,
		export.RenderSessionMarkdown, export.RenderSessionJSON)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("exports/sessions/%s.%s", sessionID, ext)
	if err := s.s3.Upload(ctx, key, content, contentType); err != nil {
		logger.Error("s3 upload failed", slog.String("key", key), slog.String("error", err.Error()))
		return "", fmt.Errorf("upload session export: %w", err)
	}

	url, err := s.s3.PresignURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("presign session export: %w", err)
	}

	logger.Info("session export uploaded",
		slog.String("session_id", sessionID),
		slog.String("format", format),
		slog.String("key", key),
	)
	return url, nil
}

// UploadAndPresignTutorial assembles the tutorial export, renders it to the
// requested format, uploads it to S3, and returns a presigned download URL.
// Returns an error if the S3 upload fails.
func (s *ExportService) UploadAndPresignTutorial(
	ctx context.Context,
	tutorialID, ownerSub, format string,
) (string, error) {
	logger := observability.LoggerFromContext(ctx)

	result, err := s.ExportTutorial(ctx, tutorialID, ownerSub)
	if err != nil {
		return "", err
	}

	content, contentType, ext, err := renderExport(result, format,
		export.RenderTutorialMarkdown, export.RenderTutorialJSON)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("exports/tutorials/%s.%s", tutorialID, ext)
	if err := s.s3.Upload(ctx, key, content, contentType); err != nil {
		logger.Error("s3 upload failed", slog.String("key", key), slog.String("error", err.Error()))
		return "", fmt.Errorf("upload tutorial export: %w", err)
	}

	url, err := s.s3.PresignURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("presign tutorial export: %w", err)
	}

	logger.Info("tutorial export uploaded",
		slog.String("tutorial_id", tutorialID),
		slog.String("format", format),
		slog.String("key", key),
	)
	return url, nil
}

// UploadAndPresignTutorialSession assembles the tutorial session export,
// renders it to the requested format, uploads it to S3, and returns a
// presigned download URL. Returns an error if the S3 upload fails.
func (s *ExportService) UploadAndPresignTutorialSession(
	ctx context.Context,
	sessionID, ownerSub, format string,
) (string, error) {
	logger := observability.LoggerFromContext(ctx)

	result, err := s.ExportTutorialSession(ctx, sessionID, ownerSub)
	if err != nil {
		return "", err
	}

	content, contentType, ext, err := renderExport(result, format,
		export.RenderTutorialSessionMarkdown, export.RenderTutorialSessionJSON)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("exports/tutorial-sessions/%s.%s", sessionID, ext)
	if err := s.s3.Upload(ctx, key, content, contentType); err != nil {
		logger.Error("s3 upload failed", slog.String("key", key), slog.String("error", err.Error()))
		return "", fmt.Errorf("upload tutorial session export: %w", err)
	}

	url, err := s.s3.PresignURL(ctx, key)
	if err != nil {
		return "", fmt.Errorf("presign tutorial session export: %w", err)
	}

	logger.Info("tutorial session export uploaded",
		slog.String("session_id", sessionID),
		slog.String("format", format),
		slog.String("key", key),
	)
	return url, nil
}

// renderExport renders the export payload to bytes using the appropriate
// renderer for the given format ("md" or "json").
// T is the export payload type (e.g. *export.SeminarExport).
func renderExport[T any](
	payload T,
	format string,
	renderMD func(T) []byte,
	renderJSON func(T) ([]byte, error),
) (content []byte, contentType, ext string, err error) {
	switch format {
	case "md":
		return renderMD(payload), "text/markdown; charset=utf-8", "md", nil
	case "json", "":
		content, err = renderJSON(payload)
		if err != nil {
			return nil, "", "", fmt.Errorf("render JSON export: %w", err)
		}
		return content, "application/json; charset=utf-8", "json", nil
	default:
		return nil, "", "", &ValidationError{
			Field:   "format",
			Message: fmt.Sprintf("unsupported export format %q: must be \"json\" or \"md\"", format),
		}
	}
}
