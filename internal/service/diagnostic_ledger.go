package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/julianstephens/formation/internal/domain"
	"github.com/julianstephens/formation/internal/repo"
)

// PatternSummaryItem represents a single pattern with its occurrence and trend data.
type PatternSummaryItem struct {
	PatternCode  string
	Occurrences  int
	LastSeenWeek string
	Trend        string // persistent, improving, resolved, emerging
}

// PatternSummary aggregates pattern data across multiple weeks.
type PatternSummary struct {
	Items []PatternSummaryItem
}

// WeekSummary provides a summary of diagnostics for the current week.
type WeekSummary struct {
	WeekOf  time.Time
	Entries []domain.DiagnosticEntry
}

// DiagnosticLedgerService handles persistence and analysis of tutorial diagnostic data.
type DiagnosticLedgerService struct {
	repo *repo.TutorialRepo
}

// NewDiagnosticLedgerService creates a new diagnostic ledger service.
func NewDiagnosticLedgerService(repo *repo.TutorialRepo) *DiagnosticLedgerService {
	return &DiagnosticLedgerService{repo: repo}
}

// RecordEntries persists multiple diagnostic entries for a tutorial session.
func (s *DiagnosticLedgerService) RecordEntries(
	ctx context.Context,
	ownerSub string,
	tutorialID string,
	tutorialSessionID string,
	weekOf time.Time,
	entries []domain.DiagnosticEntry,
) error {
	for _, entry := range entries {
		entry.TutorialID = tutorialID
		entry.TutorialSessionID = tutorialSessionID
		entry.OwnerSub = ownerSub
		entry.WeekOf = weekOf

		_, err := s.repo.CreateDiagnosticEntry(ctx, ownerSub, entry)
		if err != nil {
			return fmt.Errorf("record diagnostic entry: %w", err)
		}
	}
	return nil
}

// BuildPatternSummary computes pattern trends across the specified number of lookback weeks.
func (s *DiagnosticLedgerService) BuildPatternSummary(
	ctx context.Context,
	tutorialID, ownerSub string,
	lookbackWeeks int,
) (PatternSummary, error) {
	// Get all diagnostic entries for the tutorial
	entries, err := s.repo.ListDiagnosticEntriesByTutorial(ctx, tutorialID, ownerSub)
	if err != nil {
		return PatternSummary{}, fmt.Errorf("list diagnostic entries: %w", err)
	}

	// Filter to lookback period
	cutoff := time.Now().AddDate(0, 0, -lookbackWeeks*7)
	var recentEntries []domain.DiagnosticEntry
	for _, e := range entries {
		if e.WeekOf.After(cutoff) || e.WeekOf.Equal(cutoff) {
			recentEntries = append(recentEntries, e)
		}
	}

	// Aggregate by pattern code
	patternMap := make(map[domain.DiagnosticPatternCode]*patternAgg)
	for _, e := range recentEntries {
		if _, exists := patternMap[e.PatternCode]; !exists {
			patternMap[e.PatternCode] = &patternAgg{
				code:         e.PatternCode,
				occurrences:  0,
				lastSeenWeek: e.WeekOf,
				weeksSeen:    make(map[string]bool),
			}
		}
		agg := patternMap[e.PatternCode]
		agg.occurrences++
		if e.WeekOf.After(agg.lastSeenWeek) {
			agg.lastSeenWeek = e.WeekOf
		}
		weekKey := e.WeekOf.Format("2006-01-02")
		agg.weeksSeen[weekKey] = true
	}

	// Compute trend for each pattern
	var items []PatternSummaryItem
	for _, agg := range patternMap {
		trend := s.computeTrend(agg, recentEntries)
		items = append(items, PatternSummaryItem{
			PatternCode:  string(agg.code),
			Occurrences:  agg.occurrences,
			LastSeenWeek: agg.lastSeenWeek.Format("2006-01-02"),
			Trend:        trend,
		})
	}

	// Sort by occurrences (descending), then by last seen week (descending)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Occurrences != items[j].Occurrences {
			return items[i].Occurrences > items[j].Occurrences
		}
		return items[i].LastSeenWeek > items[j].LastSeenWeek
	})

	return PatternSummary{Items: items}, nil
}

// BuildCurrentWeekSummary returns all diagnostic entries for a specific week.
func (s *DiagnosticLedgerService) BuildCurrentWeekSummary(
	ctx context.Context,
	tutorialID, ownerSub string,
	weekOf time.Time,
) (WeekSummary, error) {
	weekStr := weekOf.Format("2006-01-02")
	entries, err := s.repo.ListDiagnosticEntriesByWeek(ctx, tutorialID, ownerSub, weekStr)
	if err != nil {
		return WeekSummary{}, fmt.Errorf("list diagnostic entries by week: %w", err)
	}

	return WeekSummary{
		WeekOf:  weekOf,
		Entries: entries,
	}, nil
}

// GetPreviousProblemSet retrieves the problem set for the week prior to the given weekOf.
func (s *DiagnosticLedgerService) GetPreviousProblemSet(
	ctx context.Context,
	tutorialID, ownerSub string,
	weekOf time.Time,
) (*domain.ProblemSet, error) {
	// Look back one week
	previousWeek := weekOf.AddDate(0, 0, -7)
	weekStr := previousWeek.Format("2006-01-02")

	ps, err := s.repo.GetProblemSetByWeek(ctx, tutorialID, ownerSub, weekStr)
	if err != nil {
		// Not finding a previous problem set is not an error, just means there isn't one
		if err == repo.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get previous problem set: %w", err)
	}

	return ps, nil
}

// RankPatternsForProblemSet scores and ranks patterns to determine which should be targeted.
// Returns the top N patterns based on persistence, severity, and recency.
func (s *DiagnosticLedgerService) RankPatternsForProblemSet(
	ctx context.Context,
	tutorialID, ownerSub string,
	topN int,
) ([]domain.DiagnosticEntry, error) {
	// Get recent entries (last 4 weeks)
	entries, err := s.repo.ListRecentDiagnosticEntries(ctx, tutorialID, ownerSub, 100)
	if err != nil {
		return nil, fmt.Errorf("list recent entries: %w", err)
	}

	// Filter to only active patterns
	var activeEntries []domain.DiagnosticEntry
	for _, e := range entries {
		if e.Status == domain.DiagnosticActive {
			activeEntries = append(activeEntries, e)
		}
	}

	// Aggregate by pattern and compute scores
	type patternScore struct {
		entry domain.DiagnosticEntry
		score int
	}

	patternScores := make(map[domain.DiagnosticPatternCode]*patternScore)
	now := time.Now()

	for _, e := range activeEntries {
		if _, exists := patternScores[e.PatternCode]; !exists {
			patternScores[e.PatternCode] = &patternScore{entry: e, score: 0}
		}

		ps := patternScores[e.PatternCode]

		// score = (occurrences * 3) + severity + recency_bonus
		ps.score += 3 // occurrence weight

		// Add severity from the most severe instance
		if e.Severity > ps.entry.Severity {
			ps.entry = e
		}

		// Recency bonus: up to 5 points if within the last week
		daysSince := now.Sub(e.WeekOf).Hours() / 24
		if daysSince < 7 {
			ps.score += 5
		} else if daysSince < 14 {
			ps.score += 3
		}
	}

	// Convert to slice and sort by score
	var scored []patternScore
	for _, ps := range patternScores {
		ps.score += ps.entry.Severity
		scored = append(scored, *ps)
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top N
	var result []domain.DiagnosticEntry
	for i := 0; i < topN && i < len(scored); i++ {
		result = append(result, scored[i].entry)
	}

	return result, nil
}

// UpdatePatternStatuses updates diagnostic entry statuses based on recent evidence.
// Should be called at the end of each extended review session.
func (s *DiagnosticLedgerService) UpdatePatternStatuses(
	ctx context.Context,
	tutorialID, ownerSub string,
	currentWeek time.Time,
) error {
	// Get entries from the last 3 weeks
	entries, err := s.repo.ListRecentDiagnosticEntries(ctx, tutorialID, ownerSub, 200)
	if err != nil {
		return fmt.Errorf("list recent entries: %w", err)
	}

	// Build pattern occurrence map by week
	type weeklyPattern struct {
		thisWeek bool
		lastWeek bool
	}

	lastWeek := currentWeek.AddDate(0, 0, -7)
	twoWeeksAgo := currentWeek.AddDate(0, 0, -14)

	patternWeeks := make(map[domain.DiagnosticPatternCode]*weeklyPattern)
	patternEntries := make(map[domain.DiagnosticPatternCode][]domain.DiagnosticEntry)

	for _, e := range entries {
		if _, exists := patternWeeks[e.PatternCode]; !exists {
			patternWeeks[e.PatternCode] = &weeklyPattern{}
		}

		patternEntries[e.PatternCode] = append(patternEntries[e.PatternCode], e)

		weekKey := e.WeekOf.Format("2006-01-02")
		currentWeekKey := currentWeek.Format("2006-01-02")
		lastWeekKey := lastWeek.Format("2006-01-02")

		if weekKey == currentWeekKey {
			patternWeeks[e.PatternCode].thisWeek = true
		} else if weekKey == lastWeekKey {
			patternWeeks[e.PatternCode].lastWeek = true
		}
	}

	// Apply status update rules
	for pattern, weeks := range patternWeeks {
		entries := patternEntries[pattern]

		var newStatus domain.DiagnosticStatus

		// Pattern seen this week again -> active
		if weeks.thisWeek {
			newStatus = domain.DiagnosticActive
		} else if weeks.lastWeek && !weeks.thisWeek {
			// Pattern not seen this week but seen last week -> improving
			newStatus = domain.DiagnosticImproving
		} else {
			// Pattern absent for 2 consecutive weeks -> resolved
			absent := true
			for _, e := range entries {
				if e.WeekOf.After(twoWeeksAgo) {
					absent = false
					break
				}
			}
			if absent {
				newStatus = domain.DiagnosticResolved
			} else {
				continue // no change
			}
		}

		// Update all entries for this pattern to the new status
		for _, e := range entries {
			if e.Status != newStatus {
				_, err := s.repo.UpdateDiagnosticEntryStatus(ctx, e.ID, ownerSub, newStatus)
				if err != nil {
					return fmt.Errorf("update diagnostic entry status: %w", err)
				}
			}
		}
	}

	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

type patternAgg struct {
	code         domain.DiagnosticPatternCode
	occurrences  int
	lastSeenWeek time.Time
	weeksSeen    map[string]bool
}

func (s *DiagnosticLedgerService) computeTrend(agg *patternAgg, allEntries []domain.DiagnosticEntry) string {
	// Simple heuristic:
	// - persistent: seen in 3+ different weeks
	// - improving: seen in only 1-2 weeks and status is improving
	// - resolved: all entries marked resolved
	// - emerging: seen in 1-2 weeks and all active

	if len(agg.weeksSeen) >= 3 {
		return "persistent"
	}

	// Check statuses of entries for this pattern
	activeCount := 0
	improvingCount := 0
	resolvedCount := 0

	for _, e := range allEntries {
		if e.PatternCode == agg.code {
			switch e.Status {
			case domain.DiagnosticActive:
				activeCount++
			case domain.DiagnosticImproving:
				improvingCount++
			case domain.DiagnosticResolved:
				resolvedCount++
			}
		}
	}

	if resolvedCount > 0 && activeCount == 0 {
		return "resolved"
	}
	if improvingCount > activeCount {
		return "improving"
	}
	if len(agg.weeksSeen) <= 2 && activeCount > 0 {
		return "emerging"
	}

	return "active"
}
