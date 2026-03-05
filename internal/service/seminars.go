// Package service contains the application-layer business logic.
// Services validate inputs, enforce invariants, and delegate persistence to
// repository types. They are intentionally free of HTTP concerns.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
)

// validModes is the exhaustive set of allowed seminar modes.
var validModes = map[string]bool{"paperback": true, "excerpt": true}

// SeminarService implements all business operations for seminars.
type SeminarService struct {
	repo *repo.SeminarRepo
}

// NewSeminarService constructs a SeminarService backed by the given repository.
func NewSeminarService(r *repo.SeminarRepo) *SeminarService {
	return &SeminarService{repo: r}
}

// ── Create ─────────────────────────────────────────────────────────────────────

// CreateParams holds all caller-supplied fields for creating a seminar.
type CreateParams struct {
	Title               string
	Author              string
	EditionNotes        string
	ThesisCurrent       string
	DefaultMode         string
	DefaultReconMinutes int
}

// Create validates params and persists a new seminar owned by ownerSub.
func (s *SeminarService) Create(ctx context.Context, ownerSub string, p CreateParams) (*domain.Seminar, error) {
	if p.DefaultMode == "" {
		p.DefaultMode = "paperback"
	}
	if p.DefaultReconMinutes == 0 {
		p.DefaultReconMinutes = 18
	}
	if err := validateSeminarFields(p.DefaultMode, p.DefaultReconMinutes); err != nil {
		return nil, err
	}
	sem := domain.Seminar{
		Title:               p.Title,
		Author:              p.Author,
		EditionNotes:        p.EditionNotes,
		ThesisCurrent:       p.ThesisCurrent,
		DefaultMode:         p.DefaultMode,
		DefaultReconMinutes: p.DefaultReconMinutes,
	}
	return s.repo.Create(ctx, ownerSub, sem)
}

// ── Get ────────────────────────────────────────────────────────────────────────

// Get returns the seminar with the given id if it is owned by ownerSub.
func (s *SeminarService) Get(ctx context.Context, id, ownerSub string) (*domain.Seminar, error) {
	sem, err := s.repo.GetByID(ctx, id, ownerSub)
	if err != nil {
		return nil, wrapNotFound(err, "seminar", id)
	}
	return sem, nil
}

// ── List ───────────────────────────────────────────────────────────────────────

// List returns all seminars owned by ownerSub.
func (s *SeminarService) List(ctx context.Context, ownerSub string) ([]domain.Seminar, error) {
	seminars, err := s.repo.List(ctx, ownerSub)
	if err != nil {
		return nil, fmt.Errorf("list seminars: %w", err)
	}
	if seminars == nil {
		seminars = []domain.Seminar{}
	}
	return seminars, nil
}

// ── Update ─────────────────────────────────────────────────────────────────────

// UpdateParams holds all patchable seminar fields; nil means "no change".
type UpdateParams struct {
	Title               *string
	Author              *string
	EditionNotes        *string
	DefaultMode         *string
	DefaultReconMinutes *int
}

// Update applies a partial update to the seminar and returns the updated record.
func (s *SeminarService) Update(ctx context.Context, id, ownerSub string, p UpdateParams) (*domain.Seminar, error) {
	if p.DefaultMode != nil && !validModes[*p.DefaultMode] {
		return nil, &ValidationError{Field: "default_mode", Message: "must be 'paperback' or 'excerpt'"}
	}
	if p.DefaultReconMinutes != nil && (*p.DefaultReconMinutes < 15 || *p.DefaultReconMinutes > 20) {
		return nil, &ValidationError{Field: "default_recon_minutes", Message: "must be between 15 and 20"}
	}
	patch := domain.SeminarPatch{
		Title:               p.Title,
		Author:              p.Author,
		EditionNotes:        p.EditionNotes,
		DefaultMode:         p.DefaultMode,
		DefaultReconMinutes: p.DefaultReconMinutes,
	}
	sem, err := s.repo.Update(ctx, id, ownerSub, patch)
	if err != nil {
		return nil, wrapNotFound(err, "seminar", id)
	}
	return sem, nil
}

// ── Delete ─────────────────────────────────────────────────────────────────────

// Delete removes the seminar. Returns ErrNotFound if it does not exist or is
// owned by a different user.
func (s *SeminarService) Delete(ctx context.Context, id, ownerSub string) error {
	if err := s.repo.Delete(ctx, id, ownerSub); err != nil {
		return wrapNotFound(err, "seminar", id)
	}
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func validateSeminarFields(mode string, reconMinutes int) error {
	if !validModes[mode] {
		return &ValidationError{Field: "default_mode", Message: "must be 'paperback' or 'excerpt'"}
	}
	if reconMinutes < 15 || reconMinutes > 20 {
		return &ValidationError{Field: "default_recon_minutes", Message: "must be between 15 and 20"}
	}
	return nil
}

// wrapNotFound converts a repo.ErrNotFound into a NotFoundError; other errors
// are returned with additional context.
func wrapNotFound(err error, resource, id string) error {
	if errors.Is(err, repo.ErrNotFound) {
		return &NotFoundError{Resource: resource, ID: id}
	}
	return fmt.Errorf("%s %s: %w", resource, id, err)
}

// ── typed errors ───────────────────────────────────────────────────────────────

// NotFoundError signals that a requested resource does not exist or is not
// accessible to the caller.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Resource, e.ID)
}

// ValidationError signals an invalid caller-supplied field value.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %q: %s", e.Field, e.Message)
}
