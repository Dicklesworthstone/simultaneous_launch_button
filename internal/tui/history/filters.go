// Package history provides TUI views for request history browsing.
package history

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
)

// TierOptions are the available tier filter options.
var TierOptions = []string{
	"",                         // All
	string(db.RiskTierCritical),
	string(db.RiskTierDangerous),
	string(db.RiskTierCaution),
}

// StatusOptions are the available status filter options.
var StatusOptions = []string{
	"",                              // All
	string(db.StatusPending),
	string(db.StatusApproved),
	string(db.StatusRejected),
	string(db.StatusExecuted),
	string(db.StatusExecutionFailed),
	string(db.StatusTimeout),
	string(db.StatusEscalated),
	string(db.StatusCancelled),
}

// Filters represents the current filter state.
type Filters struct {
	TierFilter   string
	StatusFilter string
	tierIdx      int
	statusIdx    int
}

// NewFilters creates a new filter state with no filters applied.
func NewFilters() Filters {
	return Filters{
		TierFilter:   "",
		StatusFilter: "",
		tierIdx:      0,
		statusIdx:    0,
	}
}

// CycleTier cycles through tier filter options.
func (f *Filters) CycleTier() {
	f.tierIdx = (f.tierIdx + 1) % len(TierOptions)
	f.TierFilter = TierOptions[f.tierIdx]
}

// CycleStatus cycles through status filter options.
func (f *Filters) CycleStatus() {
	f.statusIdx = (f.statusIdx + 1) % len(StatusOptions)
	f.StatusFilter = StatusOptions[f.statusIdx]
}

// SetTier sets the tier filter.
func (f *Filters) SetTier(tier string) {
	f.TierFilter = tier
	for i, t := range TierOptions {
		if t == tier {
			f.tierIdx = i
			return
		}
	}
	f.tierIdx = 0
}

// SetStatus sets the status filter.
func (f *Filters) SetStatus(status string) {
	f.StatusFilter = status
	for i, s := range StatusOptions {
		if s == status {
			f.statusIdx = i
			return
		}
	}
	f.statusIdx = 0
}

// Clear clears all filters.
func (f *Filters) Clear() {
	f.TierFilter = ""
	f.StatusFilter = ""
	f.tierIdx = 0
	f.statusIdx = 0
}

// HasFilters returns true if any filter is active.
func (f *Filters) HasFilters() bool {
	return f.TierFilter != "" || f.StatusFilter != ""
}

// RenderTierBadge renders the tier filter as a badge.
func (f *Filters) RenderTierBadge() string {
	th := theme.Current

	label := "All Tiers"
	bg := th.Surface0
	fg := th.Subtext

	if f.TierFilter != "" {
		label = f.TierFilter
		switch db.RiskTier(f.TierFilter) {
		case db.RiskTierCritical:
			bg = th.Red
			fg = th.Base
		case db.RiskTierDangerous:
			bg = th.Peach
			fg = th.Base
		case db.RiskTierCaution:
			bg = th.Yellow
			fg = th.Base
		}
	}

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Bold(f.TierFilter != "").
		Render(label)
}

// RenderStatusBadge renders the status filter as a badge.
func (f *Filters) RenderStatusBadge() string {
	th := theme.Current

	label := "All Status"
	bg := th.Surface0
	fg := th.Subtext

	if f.StatusFilter != "" {
		label = statusLabel(db.RequestStatus(f.StatusFilter))
		switch db.RequestStatus(f.StatusFilter) {
		case db.StatusApproved, db.StatusExecuted:
			bg = th.Green
			fg = th.Base
		case db.StatusRejected, db.StatusExecutionFailed:
			bg = th.Red
			fg = th.Base
		case db.StatusPending:
			bg = th.Blue
			fg = th.Base
		case db.StatusTimeout, db.StatusEscalated:
			bg = th.Yellow
			fg = th.Base
		case db.StatusCancelled:
			bg = th.Overlay0
			fg = th.Text
		}
	}

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Bold(f.StatusFilter != "").
		Render(label)
}

// statusLabel returns a human-readable label for a status.
func statusLabel(s db.RequestStatus) string {
	switch s {
	case db.StatusPending:
		return "Pending"
	case db.StatusApproved:
		return "Approved"
	case db.StatusRejected:
		return "Rejected"
	case db.StatusExecuted:
		return "Executed"
	case db.StatusExecuting:
		return "Executing"
	case db.StatusExecutionFailed:
		return "Failed"
	case db.StatusTimeout:
		return "Timeout"
	case db.StatusEscalated:
		return "Escalated"
	case db.StatusCancelled:
		return "Cancelled"
	case db.StatusTimedOut:
		return "Timed Out"
	default:
		return string(s)
	}
}
