// Package dashboard provides keyboard bindings for the dashboard.
package dashboard

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the dashboard.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding

	// Panel focus
	FocusAgents   key.Binding
	FocusRequests key.Binding
	FocusActivity key.Binding

	// Actions
	Select  key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding

	// Request actions
	Approve key.Binding
	Reject  key.Binding
	Details key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		FocusAgents: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "agents panel"),
		),
		FocusRequests: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "requests panel"),
		),
		FocusActivity: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "activity panel"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?", "h"),
			key.WithHelp("?/h", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Approve: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "approve"),
		),
		Reject: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "reject"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
	}
}

// ShortHelp returns keybindings for the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Tab,
		k.Up,
		k.Down,
		k.Select,
		k.Refresh,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns keybindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.ShiftTab},
		{k.FocusAgents, k.FocusRequests, k.FocusActivity},
		{k.Select, k.Approve, k.Reject, k.Details},
		{k.Refresh, k.Help, k.Quit},
	}
}
