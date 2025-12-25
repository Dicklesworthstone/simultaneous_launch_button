// Package theme provides theming for the SLB TUI.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines a color scheme for the TUI.
type Theme struct {
	// Primary colors
	Mauve    lipgloss.Color // Titles, accents
	Blue     lipgloss.Color // Section headers, links
	Green    lipgloss.Color // Success, approved, commands
	Yellow   lipgloss.Color // Warning, caution tier
	Red      lipgloss.Color // Error, critical tier
	Peach    lipgloss.Color // Dangerous tier
	Teal     lipgloss.Color // Info, secondary
	Pink     lipgloss.Color // Highlights
	Flamingo lipgloss.Color // Alternative accent

	// Text colors
	Text    lipgloss.Color // Normal text
	Subtext lipgloss.Color // Dimmed text

	// Surface colors
	Surface  lipgloss.Color // Panels, boxes
	Surface0 lipgloss.Color // Lighter surface
	Surface1 lipgloss.Color // Even lighter surface
	Base     lipgloss.Color // Background
	Mantle   lipgloss.Color // Darker background
	Crust    lipgloss.Color // Darkest background

	// Overlay colors
	Overlay0 lipgloss.Color
	Overlay1 lipgloss.Color
	Overlay2 lipgloss.Color

	// Meta
	Name   string
	IsDark bool
}

// FlavorName represents a Catppuccin flavor.
type FlavorName string

const (
	FlavorMocha     FlavorName = "mocha"
	FlavorMacchiato FlavorName = "macchiato"
	FlavorFrappe    FlavorName = "frappe"
	FlavorLatte     FlavorName = "latte"
)

// Current holds the active theme.
var Current = Mocha()

// SetTheme sets the current theme by flavor name.
func SetTheme(flavor FlavorName) {
	switch flavor {
	case FlavorMocha:
		Current = Mocha()
	case FlavorMacchiato:
		Current = Macchiato()
	case FlavorFrappe:
		Current = Frappe()
	case FlavorLatte:
		Current = Latte()
	default:
		Current = Mocha()
	}
}

// TierColor returns the color for a risk tier.
func (t *Theme) TierColor(tier string) lipgloss.Color {
	switch tier {
	case "critical", "CRITICAL":
		return t.Red
	case "dangerous", "DANGEROUS":
		return t.Peach
	case "caution", "CAUTION":
		return t.Yellow
	case "safe", "SAFE":
		return t.Green
	default:
		return t.Text
	}
}

// StatusColor returns the color for a request status.
func (t *Theme) StatusColor(status string) lipgloss.Color {
	switch status {
	case "pending", "PENDING":
		return t.Blue
	case "approved", "APPROVED":
		return t.Green
	case "rejected", "REJECTED":
		return t.Red
	case "executed", "EXECUTED":
		return t.Green // Will be dimmed in style
	case "failed", "FAILED":
		return t.Red // Will be dimmed in style
	case "timeout", "TIMEOUT":
		return t.Yellow
	case "cancelled", "CANCELLED":
		return t.Subtext
	case "escalated", "ESCALATED":
		return t.Peach
	default:
		return t.Text
	}
}

// TierEmoji returns the emoji for a risk tier.
func TierEmoji(tier string) string {
	switch tier {
	case "critical", "CRITICAL":
		return "🔴"
	case "dangerous", "DANGEROUS":
		return "🟠"
	case "caution", "CAUTION":
		return "🟡"
	case "safe", "SAFE":
		return "🟢"
	default:
		return "⚪"
	}
}

// StatusIcon returns the icon for a request status.
func StatusIcon(status string) string {
	switch status {
	case "pending", "PENDING":
		return "⏳"
	case "approved", "APPROVED":
		return "✓"
	case "rejected", "REJECTED":
		return "✗"
	case "executed", "EXECUTED":
		return "✓"
	case "failed", "FAILED":
		return "✗"
	case "timeout", "TIMEOUT":
		return "⏰"
	case "cancelled", "CANCELLED":
		return "⊘"
	case "escalated", "ESCALATED":
		return "⚠"
	default:
		return "?"
	}
}
