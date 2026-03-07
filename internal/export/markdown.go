package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/julianstephens/formation/internal/domain"
)

// RenderSeminarMarkdown produces a human-readable Markdown document for a
// full seminar export including all sessions and their transcripts.
func RenderSeminarMarkdown(e *SeminarExport) []byte {
	var sb strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	fmt.Fprintf(&sb, "# Seminar Export: %s\n\n", e.Seminar.Title)
	if e.Seminar.Author != "" {
		fmt.Fprintf(&sb, "**Author:** %s  \n", e.Seminar.Author)
	}
	if e.Seminar.EditionNotes != "" {
		fmt.Fprintf(&sb, "**Edition notes:** %s  \n", e.Seminar.EditionNotes)
	}
	fmt.Fprintf(&sb, "**Thesis:** %s  \n", e.Seminar.ThesisCurrent)
	fmt.Fprintf(&sb, "**Default mode:** %s  \n", e.Seminar.DefaultMode)
	fmt.Fprintf(&sb, "**Default recon minutes:** %d  \n", e.Seminar.DefaultReconMinutes)
	fmt.Fprintf(&sb, "**Created:** %s  \n\n", fmtTime(e.Seminar.CreatedAt))

	// ── Sessions ──────────────────────────────────────────────────────────────
	if len(e.Sessions) > 0 {
		fmt.Fprintf(&sb, "## Sessions (%d)\n\n", len(e.Sessions))
		for i, se := range e.Sessions {
			renderSessionSection(&sb, i+1, &se)
		}
	} else {
		sb.WriteString("## Sessions\n\n_No sessions recorded._\n")
	}

	return []byte(sb.String())
}

// RenderSessionMarkdown produces a human-readable Markdown document for a
// single session export including its full transcript.
func RenderSessionMarkdown(e *SessionExport) []byte {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Session Export: %s\n\n", e.Session.SectionLabel)
	renderSessionSection(&sb, 0, e)

	return []byte(sb.String())
}

// ── helpers ───────────────────────────────────────────────────────────────────

func renderSessionSection(sb *strings.Builder, n int, se *SessionExport) {
	s := se.Session

	// Section heading: numbered when called from seminar export, plain otherwise.
	if n > 0 {
		fmt.Fprintf(sb, "### %d. %s\n\n", n, s.SectionLabel)
	}

	fmt.Fprintf(sb, "**Mode:** %s  \n", s.Mode)
	if s.ExcerptText != "" {
		fmt.Fprintf(sb, "**Excerpt:**\n\n> %s\n\n", strings.ReplaceAll(s.ExcerptText, "\n", "\n> "))
	}
	fmt.Fprintf(sb, "**Status:** %s  \n", s.Status)
	fmt.Fprintf(sb, "**Phase:** %s  \n", s.Phase)
	fmt.Fprintf(sb, "**Recon minutes:** %d  \n", s.ReconMinutes)
	fmt.Fprintf(sb, "**Started:** %s  \n", fmtTime(s.StartedAt))
	if s.EndedAt != nil {
		fmt.Fprintf(sb, "**Ended:** %s  \n", fmtTime(*s.EndedAt))
	}
	if s.ResidueText != "" {
		fmt.Fprintf(sb, "\n**Residue statement:**\n\n> %s\n", s.ResidueText)
	}

	sb.WriteString("\n#### Transcript\n\n")

	if len(se.Turns) == 0 {
		sb.WriteString("_No turns recorded._\n\n")
		return
	}

	var lastPhase domain.SessionPhase
	for _, t := range se.Turns {
		// Emit a phase break heading whenever the phase changes.
		if t.Phase != lastPhase {
			fmt.Fprintf(sb, "\n**Phase: %s**\n\n", t.Phase)
			lastPhase = t.Phase
		}

		speaker := speakerLabel(t.Speaker)
		fmt.Fprintf(sb, "**%s** _%s_\n\n%s\n\n",
			speaker, fmtTime(t.CreatedAt), t.Text)

		if len(t.Flags) > 0 {
			fmt.Fprintf(sb, "_Flags: %s_\n\n", strings.Join(t.Flags, ", "))
		}
	}
}

func speakerLabel(speaker string) string {
	switch speaker {
	case "agent":
		return "Tutor"
	case "system":
		return "System"
	default:
		return "You"
	}
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format("2006-01-02 15:04 UTC")
}

// RenderTutorialMarkdown produces a human-readable Markdown document for a
// full tutorial export including all sessions and their transcripts.
func RenderTutorialMarkdown(e *TutorialExport) []byte {
	var sb strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	fmt.Fprintf(&sb, "# Tutorial Export: %s\n\n", e.Tutorial.Title)
	fmt.Fprintf(&sb, "**Subject:** %s  \n", e.Tutorial.Subject)
	if e.Tutorial.Description != "" {
		fmt.Fprintf(&sb, "**Description:** %s  \n", e.Tutorial.Description)
	}
	fmt.Fprintf(&sb, "**Difficulty:** %s  \n", e.Tutorial.Difficulty)
	fmt.Fprintf(&sb, "**Created:** %s  \n\n", fmtTime(e.Tutorial.CreatedAt))

	// ── Sessions ──────────────────────────────────────────────────────────────
	if len(e.Sessions) > 0 {
		fmt.Fprintf(&sb, "## Sessions (%d)\n\n", len(e.Sessions))
		for i, se := range e.Sessions {
			renderTutorialSessionSection(&sb, i+1, &se)
		}
	} else {
		sb.WriteString("## Sessions\n\n_No sessions recorded._\n")
	}

	return []byte(sb.String())
}

// RenderTutorialSessionMarkdown produces a human-readable Markdown document for a
// single tutorial session export including its full transcript.
func RenderTutorialSessionMarkdown(e *TutorialSessionExport) []byte {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Tutorial Session Export\n\n")
	renderTutorialSessionSection(&sb, 0, e)

	return []byte(sb.String())
}

// ── tutorial helpers ──────────────────────────────────────────────────────────

func renderTutorialSessionSection(sb *strings.Builder, n int, se *TutorialSessionExport) {
	s := se.Session

	// Section heading: numbered when called from tutorial export, plain otherwise.
	if n > 0 {
		fmt.Fprintf(sb, "### %d. Tutorial Session\n\n", n)
	}

	fmt.Fprintf(sb, "**Session ID:** %s  \n", s.ID)
	fmt.Fprintf(sb, "**Status:** %s  \n", s.Status)
	if s.Kind != "" {
		fmt.Fprintf(sb, "**Kind:** %s  \n", s.Kind)
	}
	if s.Notes != "" {
		fmt.Fprintf(sb, "**Notes:** %s  \n", s.Notes)
	}
	fmt.Fprintf(sb, "**Started:** %s  \n", fmtTime(s.StartedAt))
	if s.EndedAt != nil {
		fmt.Fprintf(sb, "**Ended:** %s  \n", fmtTime(*s.EndedAt))
	}

	// ── Artifacts ─────────────────────────────────────────────────────────────
	if len(se.Artifacts) > 0 {
		sb.WriteString("\n#### Artifacts\n\n")
		for _, a := range se.Artifacts {
			fmt.Fprintf(sb, "**%s** _%s_  \n", a.Title, a.Kind)
			fmt.Fprintf(sb, "_%s_\n\n", fmtTime(a.CreatedAt))
			if a.Content != "" {
				fmt.Fprintf(sb, "```\n%s\n```\n\n", a.Content)
			}
		}
	}

	sb.WriteString("\n#### Transcript\n\n")

	if len(se.Turns) == 0 {
		sb.WriteString("_No turns recorded._\n\n")
		return
	}

	for _, t := range se.Turns {
		speaker := tutorialSpeakerLabel(t.Speaker)
		fmt.Fprintf(sb, "**%s** _%s_\n\n%s\n\n",
			speaker, fmtTime(t.CreatedAt), t.Text)
	}
}

func tutorialSpeakerLabel(speaker string) string {
	switch speaker {
	case "agent":
		return "Tutor"
	case "system":
		return "System"
	default:
		return "You"
	}
}
