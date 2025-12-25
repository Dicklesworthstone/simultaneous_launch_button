// Package components provides timeline components for request lifecycle.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// TimelineEvent represents a single event in the request timeline.
type TimelineEvent struct {
	State     string
	Timestamp time.Time
	Actor     string // Who performed this action
	Details   string // Additional details
}

// Timeline renders a request lifecycle timeline.
type Timeline struct {
	Events   []TimelineEvent
	Compact  bool
	Expanded bool
	Current  string // Current state to highlight
}

// NewTimeline creates a new timeline component.
func NewTimeline() *Timeline {
	return &Timeline{}
}

// AddEvent adds an event to the timeline.
func (t *Timeline) AddEvent(state string, ts time.Time, actor, details string) *Timeline {
	t.Events = append(t.Events, TimelineEvent{
		State:     state,
		Timestamp: ts,
		Actor:     actor,
		Details:   details,
	})
	return t
}

// WithCurrent sets the current state to highlight.
func (t *Timeline) WithCurrent(state string) *Timeline {
	t.Current = state
	return t
}

// AsCompact sets the timeline to compact mode.
func (t *Timeline) AsCompact() *Timeline {
	t.Compact = true
	return t
}

// AsExpanded sets the timeline to expanded mode.
func (t *Timeline) AsExpanded() *Timeline {
	t.Expanded = true
	return t
}

// Render renders the timeline.
func (t *Timeline) Render() string {
	if t.Compact {
		return t.renderCompact()
	}
	if t.Expanded {
		return t.renderExpanded()
	}
	return t.renderNormal()
}

// renderCompact renders a single-line compact timeline.
func (t *Timeline) renderCompact() string {
	th := theme.Current

	// State order
	states := []string{"created", "pending", "approved", "executing", "executed"}

	var parts []string
	activeIdx := -1

	for i, state := range states {
		// Check if this state has been reached
		reached := t.hasReachedState(state)

		var color lipgloss.Color
		if state == t.Current {
			color = th.Mauve
			activeIdx = i
		} else if reached {
			color = th.Green
		} else {
			color = th.Overlay0
		}

		dot := lipgloss.NewStyle().Foreground(color).Render("●")
		parts = append(parts, dot)
	}

	// Connect with arrows
	result := ""
	for i, part := range parts {
		if i > 0 {
			arrowColor := th.Overlay0
			if activeIdx >= 0 && i <= activeIdx {
				arrowColor = th.Green
			}
			arrow := lipgloss.NewStyle().Foreground(arrowColor).Render(" → ")
			result += arrow
		}
		result += part
	}

	return result
}

// renderNormal renders the standard timeline view.
func (t *Timeline) renderNormal() string {
	th := theme.Current

	var lines []string

	for i, event := range t.Events {
		isLast := i == len(t.Events)-1
		isCurrent := event.State == t.Current

		// State indicator
		var stateColor lipgloss.Color
		switch strings.ToLower(event.State) {
		case "approved", "executed":
			stateColor = th.Green
		case "rejected", "failed":
			stateColor = th.Red
		case "pending", "executing":
			stateColor = th.Blue
		case "timeout", "escalated":
			stateColor = th.Yellow
		default:
			stateColor = th.Subtext
		}

		// Build the line
		connector := "│"
		node := "●"
		if isLast {
			connector = " "
		}
		if isCurrent {
			node = "◉"
		}

		nodeStyle := lipgloss.NewStyle().Foreground(stateColor).Bold(isCurrent)
		connectorStyle := lipgloss.NewStyle().Foreground(th.Overlay0)

		stateLabel := lipgloss.NewStyle().
			Foreground(stateColor).
			Bold(isCurrent).
			Render(strings.ToUpper(event.State))

		timeStr := ""
		if !event.Timestamp.IsZero() {
			timeStr = lipgloss.NewStyle().
				Foreground(th.Subtext).
				Render("  " + event.Timestamp.Format("15:04:05"))
		}

		line := fmt.Sprintf("%s %s%s",
			nodeStyle.Render(node),
			stateLabel,
			timeStr,
		)
		lines = append(lines, line)

		// Add connector to next event
		if !isLast {
			lines = append(lines, connectorStyle.Render(connector))
		}
	}

	return strings.Join(lines, "\n")
}

// renderExpanded renders the full expanded timeline with details.
func (t *Timeline) renderExpanded() string {
	th := theme.Current

	var lines []string

	for i, event := range t.Events {
		isLast := i == len(t.Events)-1
		isCurrent := event.State == t.Current

		// State indicator
		var stateColor lipgloss.Color
		switch strings.ToLower(event.State) {
		case "approved", "executed":
			stateColor = th.Green
		case "rejected", "failed":
			stateColor = th.Red
		case "pending", "executing":
			stateColor = th.Blue
		case "timeout", "escalated":
			stateColor = th.Yellow
		default:
			stateColor = th.Subtext
		}

		nodeStyle := lipgloss.NewStyle().Foreground(stateColor).Bold(isCurrent)
		connectorStyle := lipgloss.NewStyle().Foreground(th.Overlay0)

		node := "●"
		if isCurrent {
			node = "◉"
		}

		stateLabel := lipgloss.NewStyle().
			Foreground(stateColor).
			Bold(isCurrent).
			Render(strings.ToUpper(event.State))

		// Main line
		line := fmt.Sprintf("%s %s", nodeStyle.Render(node), stateLabel)
		lines = append(lines, line)

		// Details (indented)
		if !event.Timestamp.IsZero() {
			timeStr := event.Timestamp.Format("2006-01-02 15:04:05")
			lines = append(lines, connectorStyle.Render("│  ")+
				lipgloss.NewStyle().Foreground(th.Subtext).Render(timeStr))
		}

		if event.Actor != "" {
			lines = append(lines, connectorStyle.Render("│  ")+
				lipgloss.NewStyle().Foreground(th.Subtext).Render("by "+event.Actor))
		}

		if event.Details != "" {
			lines = append(lines, connectorStyle.Render("│  ")+
				lipgloss.NewStyle().Foreground(th.Text).Render(event.Details))
		}

		// Connector to next
		if !isLast {
			lines = append(lines, connectorStyle.Render("│"))
		}
	}

	return strings.Join(lines, "\n")
}

// hasReachedState checks if the timeline has reached a given state.
func (t *Timeline) hasReachedState(state string) bool {
	for _, event := range t.Events {
		if strings.EqualFold(event.State, state) {
			return true
		}
	}
	return false
}

// RenderTimeline is a convenience function to create and render a timeline.
func RenderTimeline(events []TimelineEvent, current string) string {
	tl := NewTimeline().WithCurrent(current)
	for _, e := range events {
		tl.AddEvent(e.State, e.Timestamp, e.Actor, e.Details)
	}
	return tl.Render()
}

// RenderTimelineCompact is a convenience function for compact timeline.
func RenderTimelineCompact(events []TimelineEvent, current string) string {
	tl := NewTimeline().WithCurrent(current).AsCompact()
	for _, e := range events {
		tl.AddEvent(e.State, e.Timestamp, e.Actor, e.Details)
	}
	return tl.Render()
}
