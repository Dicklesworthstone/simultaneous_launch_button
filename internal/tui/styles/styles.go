// Package styles provides reusable lipgloss styles for the SLB TUI.
package styles

import (
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the styled lipgloss renderers.
type Styles struct {
	// Title styles
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	SectionHead lipgloss.Style

	// Text styles
	Normal    lipgloss.Style
	Dimmed    lipgloss.Style
	Bold      lipgloss.Style
	Highlight lipgloss.Style

	// Status badge styles
	BadgePending   lipgloss.Style
	BadgeApproved  lipgloss.Style
	BadgeRejected  lipgloss.Style
	BadgeExecuted  lipgloss.Style
	BadgeFailed    lipgloss.Style
	BadgeTimeout   lipgloss.Style
	BadgeCancelled lipgloss.Style
	BadgeEscalated lipgloss.Style

	// Tier badge styles
	TierCritical  lipgloss.Style
	TierDangerous lipgloss.Style
	TierCaution   lipgloss.Style
	TierSafe      lipgloss.Style

	// Container styles
	Panel      lipgloss.Style
	CommandBox lipgloss.Style
	Card       lipgloss.Style
	Selected   lipgloss.Style

	// Layout helpers
	Border   lipgloss.Style
	NoBorder lipgloss.Style
	Padded   lipgloss.Style
	Centered lipgloss.Style
}

// New creates a new Styles instance from the current theme.
func New() *Styles {
	return FromTheme(theme.Current)
}

// FromTheme creates styles from a specific theme.
func FromTheme(t *theme.Theme) *Styles {
	s := &Styles{}

	// Title styles
	s.Title = lipgloss.NewStyle().
		Foreground(t.Mauve).
		Bold(true)

	s.Subtitle = lipgloss.NewStyle().
		Foreground(t.Subtext).
		Italic(true)

	s.SectionHead = lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	// Text styles
	s.Normal = lipgloss.NewStyle().
		Foreground(t.Text)

	s.Dimmed = lipgloss.NewStyle().
		Foreground(t.Subtext)

	s.Bold = lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true)

	s.Highlight = lipgloss.NewStyle().
		Foreground(t.Pink).
		Bold(true)

	// Badge base style
	badgeBase := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	// Status badge styles
	s.BadgePending = badgeBase.
		Foreground(t.Base).
		Background(t.Blue)

	s.BadgeApproved = badgeBase.
		Foreground(t.Base).
		Background(t.Green)

	s.BadgeRejected = badgeBase.
		Foreground(t.Base).
		Background(t.Red)

	s.BadgeExecuted = badgeBase.
		Foreground(t.Base).
		Background(t.Green)

	s.BadgeFailed = badgeBase.
		Foreground(t.Base).
		Background(t.Red)

	s.BadgeTimeout = badgeBase.
		Foreground(t.Base).
		Background(t.Yellow)

	s.BadgeCancelled = badgeBase.
		Foreground(t.Text).
		Background(t.Overlay0)

	s.BadgeEscalated = badgeBase.
		Foreground(t.Base).
		Background(t.Peach)

	// Tier badge styles
	s.TierCritical = badgeBase.
		Foreground(t.Base).
		Background(t.Red)

	s.TierDangerous = badgeBase.
		Foreground(t.Base).
		Background(t.Peach)

	s.TierCaution = badgeBase.
		Foreground(t.Base).
		Background(t.Yellow)

	s.TierSafe = badgeBase.
		Foreground(t.Base).
		Background(t.Green)

	// Container styles
	s.Panel = lipgloss.NewStyle().
		Background(t.Surface).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay0)

	s.CommandBox = lipgloss.NewStyle().
		Background(t.Mantle).
		Foreground(t.Green).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay0)

	s.Card = lipgloss.NewStyle().
		Background(t.Surface0).
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Overlay0)

	s.Selected = lipgloss.NewStyle().
		Background(t.Surface1).
		Border(lipgloss.ThickBorder()).
		BorderForeground(t.Mauve)

	// Layout helpers
	s.Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay0)

	s.NoBorder = lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder())

	s.Padded = lipgloss.NewStyle().
		Padding(1, 2)

	s.Centered = lipgloss.NewStyle().
		Align(lipgloss.Center)

	return s
}

// StatusBadge returns the appropriate badge style for a status.
func (s *Styles) StatusBadge(status string) lipgloss.Style {
	switch status {
	case "pending", "PENDING":
		return s.BadgePending
	case "approved", "APPROVED":
		return s.BadgeApproved
	case "rejected", "REJECTED":
		return s.BadgeRejected
	case "executed", "EXECUTED":
		return s.BadgeExecuted
	case "failed", "FAILED":
		return s.BadgeFailed
	case "timeout", "TIMEOUT":
		return s.BadgeTimeout
	case "cancelled", "CANCELLED":
		return s.BadgeCancelled
	case "escalated", "ESCALATED":
		return s.BadgeEscalated
	default:
		return s.Dimmed
	}
}

// TierBadge returns the appropriate badge style for a tier.
func (s *Styles) TierBadge(tier string) lipgloss.Style {
	switch tier {
	case "critical", "CRITICAL":
		return s.TierCritical
	case "dangerous", "DANGEROUS":
		return s.TierDangerous
	case "caution", "CAUTION":
		return s.TierCaution
	case "safe", "SAFE":
		return s.TierSafe
	default:
		return s.Dimmed
	}
}

// RenderStatusBadge renders a status as a styled badge.
func (s *Styles) RenderStatusBadge(status string) string {
	icon := theme.StatusIcon(status)
	return s.StatusBadge(status).Render(icon + " " + status)
}

// RenderTierBadge renders a tier as a styled badge.
func (s *Styles) RenderTierBadge(tier string) string {
	emoji := theme.TierEmoji(tier)
	return s.TierBadge(tier).Render(emoji + " " + tier)
}
