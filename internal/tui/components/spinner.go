// Package components provides spinner components.
package components

import (
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerStyle defines different spinner animation styles.
type SpinnerStyle int

const (
	SpinnerStyleDots SpinnerStyle = iota
	SpinnerStyleLine
	SpinnerStyleMiniDot
	SpinnerStyleJump
	SpinnerStylePulse
	SpinnerStylePoints
	SpinnerStyleGlobe
	SpinnerStyleMoon
	SpinnerStyleMonkey
	SpinnerStyleMeter
	SpinnerStyleHamburger
)

// NewSpinner creates a new spinner with the given style.
func NewSpinner(style SpinnerStyle) spinner.Model {
	s := spinner.New()

	// Set spinner type based on style
	switch style {
	case SpinnerStyleDots:
		s.Spinner = spinner.Dot
	case SpinnerStyleLine:
		s.Spinner = spinner.Line
	case SpinnerStyleMiniDot:
		s.Spinner = spinner.MiniDot
	case SpinnerStyleJump:
		s.Spinner = spinner.Jump
	case SpinnerStylePulse:
		s.Spinner = spinner.Pulse
	case SpinnerStylePoints:
		s.Spinner = spinner.Points
	case SpinnerStyleGlobe:
		s.Spinner = spinner.Globe
	case SpinnerStyleMoon:
		s.Spinner = spinner.Moon
	case SpinnerStyleMonkey:
		s.Spinner = spinner.Monkey
	case SpinnerStyleMeter:
		s.Spinner = spinner.Meter
	case SpinnerStyleHamburger:
		s.Spinner = spinner.Hamburger
	default:
		s.Spinner = spinner.Dot
	}

	// Apply theme color
	t := theme.Current
	s.Style = lipgloss.NewStyle().Foreground(t.Mauve)

	return s
}

// DefaultSpinner creates the default spinner.
func DefaultSpinner() spinner.Model {
	return NewSpinner(SpinnerStyleDots)
}

// LoadingSpinner creates a spinner for loading states.
func LoadingSpinner() spinner.Model {
	s := NewSpinner(SpinnerStyleDots)
	t := theme.Current
	s.Style = lipgloss.NewStyle().Foreground(t.Blue)
	return s
}

// ProcessingSpinner creates a spinner for processing states.
func ProcessingSpinner() spinner.Model {
	s := NewSpinner(SpinnerStyleLine)
	t := theme.Current
	s.Style = lipgloss.NewStyle().Foreground(t.Green)
	return s
}

// WaitingSpinner creates a spinner for waiting states.
func WaitingSpinner() spinner.Model {
	s := NewSpinner(SpinnerStylePulse)
	t := theme.Current
	s.Style = lipgloss.NewStyle().Foreground(t.Yellow)
	return s
}

// SpinnerWithLabel renders a spinner with a label.
func SpinnerWithLabel(s spinner.Model, label string) string {
	t := theme.Current
	labelStyle := lipgloss.NewStyle().Foreground(t.Text)
	return s.View() + " " + labelStyle.Render(label)
}
