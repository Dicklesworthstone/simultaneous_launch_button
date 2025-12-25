// Package styles provides shimmer/glow effects for the TUI.
package styles

import (
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ShimmerState represents the current state of a shimmer animation.
type ShimmerState struct {
	Position int
	Width    int
	Forward  bool
}

// NewShimmerState creates a new shimmer animation state.
func NewShimmerState(width int) *ShimmerState {
	return &ShimmerState{
		Position: 0,
		Width:    width,
		Forward:  true,
	}
}

// Advance moves the shimmer position by one step.
// Returns true if the animation completed a full cycle.
func (s *ShimmerState) Advance() bool {
	if s.Forward {
		s.Position++
		if s.Position >= s.Width {
			s.Forward = false
			return true
		}
	} else {
		s.Position--
		if s.Position <= 0 {
			s.Forward = true
		}
	}
	return false
}

// Reset resets the shimmer to the beginning.
func (s *ShimmerState) Reset() {
	s.Position = 0
	s.Forward = true
}

// RenderShimmer applies a shimmer effect to text at the current position.
func (s *ShimmerState) RenderShimmer(text string, highlightColor lipgloss.Color) string {
	t := theme.Current
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	// Adjust width if text is shorter
	width := s.Width
	if len(runes) < width {
		width = len(runes)
	}

	result := ""
	shimmerWidth := 3 // Width of the shimmer highlight

	for i, r := range runes {
		var style lipgloss.Style

		// Calculate distance from shimmer center
		distance := abs(i - s.Position)
		if distance < shimmerWidth {
			// Within shimmer range - apply highlight
			style = lipgloss.NewStyle().Foreground(highlightColor).Bold(true)
		} else {
			// Normal text
			style = lipgloss.NewStyle().Foreground(t.Text)
		}

		result += style.Render(string(r))
	}

	return result
}

// GlowStyle creates a style with a "glow" effect using the surface colors.
func GlowStyle(baseColor lipgloss.Color) lipgloss.Style {
	t := theme.Current
	return lipgloss.NewStyle().
		Foreground(baseColor).
		Background(t.Surface).
		Bold(true).
		Padding(0, 1)
}

// FocusGlow returns a glowing style for focused elements.
func FocusGlow() lipgloss.Style {
	t := theme.Current
	return GlowStyle(t.Mauve)
}

// SuccessGlow returns a glowing style for success states.
func SuccessGlow() lipgloss.Style {
	t := theme.Current
	return GlowStyle(t.Green)
}

// WarningGlow returns a glowing style for warning states.
func WarningGlow() lipgloss.Style {
	t := theme.Current
	return GlowStyle(t.Yellow)
}

// ErrorGlow returns a glowing style for error states.
func ErrorGlow() lipgloss.Style {
	t := theme.Current
	return GlowStyle(t.Red)
}

// abs returns the absolute value of an integer.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
