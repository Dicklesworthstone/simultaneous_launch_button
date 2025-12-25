// Package styles provides gradient text effects for the TUI.
package styles

import (
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Gradient represents a color gradient for text.
type Gradient struct {
	Colors []lipgloss.Color
}

// NewGradient creates a gradient from the given colors.
func NewGradient(colors ...lipgloss.Color) *Gradient {
	return &Gradient{Colors: colors}
}

// MauveBlueGradient returns a mauve-to-blue gradient.
func MauveBlueGradient() *Gradient {
	t := theme.Current
	return NewGradient(t.Mauve, t.Pink, t.Blue)
}

// RainbowGradient returns a rainbow gradient using theme colors.
func RainbowGradient() *Gradient {
	t := theme.Current
	return NewGradient(t.Red, t.Peach, t.Yellow, t.Green, t.Teal, t.Blue, t.Mauve)
}

// TierGradient returns a gradient representing all risk tiers.
func TierGradient() *Gradient {
	t := theme.Current
	return NewGradient(t.Green, t.Yellow, t.Peach, t.Red)
}

// Render applies the gradient to a string.
// Characters are colored according to their position in the gradient.
func (g *Gradient) Render(s string) string {
	if len(g.Colors) == 0 || len(s) == 0 {
		return s
	}

	runes := []rune(s)
	result := ""

	for i, r := range runes {
		// Calculate which color to use based on position
		colorIdx := (i * (len(g.Colors) - 1)) / max(len(runes)-1, 1)
		if colorIdx >= len(g.Colors) {
			colorIdx = len(g.Colors) - 1
		}

		style := lipgloss.NewStyle().Foreground(g.Colors[colorIdx])
		result += style.Render(string(r))
	}

	return result
}

// RenderInterpolated renders text with smooth color interpolation.
// This creates a more gradual transition between colors.
func (g *Gradient) RenderInterpolated(s string) string {
	if len(g.Colors) < 2 || len(s) == 0 {
		return g.Render(s) // Fallback to simple render
	}

	// For now, use the simple render. True interpolation would require
	// parsing hex colors and blending, which adds complexity.
	// The simple stepped gradient is visually appealing for most uses.
	return g.Render(s)
}

// GradientTitle renders a title with gradient styling.
func GradientTitle(text string) string {
	return MauveBlueGradient().Render(text)
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
