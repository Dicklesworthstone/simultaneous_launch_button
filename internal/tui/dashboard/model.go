// Package dashboard implements the main TUI dashboard view.
package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Dicklesworthstone/slb/internal/tui/components"
	"github.com/Dicklesworthstone/slb/internal/tui/icons"
	"github.com/Dicklesworthstone/slb/internal/tui/styles"
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
)

// FocusPanel identifies which panel has focus.
type FocusPanel int

const (
	FocusAgents FocusPanel = iota
	FocusRequests
	FocusActivity
)

// DaemonStatus represents the daemon connection status.
type DaemonStatus int

const (
	DaemonUnknown DaemonStatus = iota
	DaemonConnected
	DaemonDisconnected
)

// AgentData holds agent display information.
type AgentData struct {
	Info     components.AgentInfo
	Selected bool
}

// RequestData holds request display information.
type RequestData struct {
	ID           string
	Command      string
	Tier         string
	Status       string
	RequestorID  string
	CreatedAt    time.Time
	Selected     bool
}

// ActivityItem represents a recent activity entry.
type ActivityItem struct {
	Action     string
	Actor      string
	RequestID  string
	Timestamp  time.Time
}

// Model is the main dashboard Bubble Tea model.
type Model struct {
	// State
	ready        bool
	width        int
	height       int
	focus        FocusPanel
	daemonStatus DaemonStatus

	// Data
	agents         []AgentData
	requests       []RequestData
	activities     []ActivityItem
	agentCursor    int
	requestCursor  int
	activityCursor int

	// Stats
	approvedCount24h int
	rejectedCount24h int
	avgResponseSec   float64

	// Styling
	styles *styles.Styles
}

// New creates a new dashboard model.
func New() Model {
	return Model{
		focus:        FocusRequests,
		daemonStatus: DaemonUnknown,
		styles:       styles.New(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

// tickMsg is sent periodically to update the dashboard.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tickMsg:
		// Periodic refresh
		return m, tickCmd()
	}

	return m, nil
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Quit
	case "q", "ctrl+c":
		return m, tea.Quit

	// Panel navigation
	case "tab":
		m.focus = (m.focus + 1) % 3
	case "shift+tab":
		m.focus = (m.focus + 2) % 3

	// Focus specific panels
	case "1":
		m.focus = FocusAgents
	case "2":
		m.focus = FocusRequests
	case "3":
		m.focus = FocusActivity

	// Cursor navigation
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()

	// Actions
	case "enter":
		return m.handleSelect()

	// Refresh
	case "r":
		// Trigger refresh
		return m, nil

	// Help
	case "?", "h":
		// TODO: Show help overlay
		return m, nil
	}

	return m, nil
}

// moveCursorUp moves the cursor up in the focused panel.
func (m *Model) moveCursorUp() {
	switch m.focus {
	case FocusAgents:
		if m.agentCursor > 0 {
			m.agentCursor--
		}
	case FocusRequests:
		if m.requestCursor > 0 {
			m.requestCursor--
		}
	case FocusActivity:
		if m.activityCursor > 0 {
			m.activityCursor--
		}
	}
}

// moveCursorDown moves the cursor down in the focused panel.
func (m *Model) moveCursorDown() {
	switch m.focus {
	case FocusAgents:
		if m.agentCursor < len(m.agents)-1 {
			m.agentCursor++
		}
	case FocusRequests:
		if m.requestCursor < len(m.requests)-1 {
			m.requestCursor++
		}
	case FocusActivity:
		if m.activityCursor < len(m.activities)-1 {
			m.activityCursor++
		}
	}
}

// handleSelect handles the enter key on the focused panel.
func (m Model) handleSelect() (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusAgents:
		// TODO: Show agent details
	case FocusRequests:
		// TODO: Show request details
	case FocusActivity:
		// TODO: Show activity details
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading dashboard..."
	}

	t := theme.Current
	ic := icons.Current()

	// Calculate panel dimensions
	headerHeight := 3
	footerHeight := 2
	contentHeight := m.height - headerHeight - footerHeight

	leftWidth := m.width / 4
	rightWidth := m.width / 4
	centerWidth := m.width - leftWidth - rightWidth - 4 // account for borders

	// Render header
	header := m.renderHeader(ic, t)

	// Render panels
	agentsPanel := m.renderAgentsPanel(leftWidth, contentHeight, t, ic)
	requestsPanel := m.renderRequestsPanel(centerWidth, contentHeight, t, ic)
	activityPanel := m.renderActivityPanel(rightWidth, contentHeight, t, ic)

	// Combine panels horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		agentsPanel,
		requestsPanel,
		activityPanel,
	)

	// Render footer
	footer := m.renderFooter(t, ic)

	// Combine vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the dashboard header bar.
func (m Model) renderHeader(ic *icons.IconSet, t *theme.Theme) string {
	// Title
	title := lipgloss.NewStyle().
		Foreground(t.Mauve).
		Bold(true).
		Render("SLB Dashboard")

	// Daemon status
	var statusIcon, statusText string
	var statusColor lipgloss.Color
	switch m.daemonStatus {
	case DaemonConnected:
		statusIcon = "●"
		statusText = "Daemon Running"
		statusColor = t.Green
	case DaemonDisconnected:
		statusIcon = "●"
		statusText = "Daemon Disconnected"
		statusColor = t.Red
	default:
		statusIcon = "●"
		statusText = "Checking..."
		statusColor = t.Yellow
	}

	status := lipgloss.NewStyle().
		Foreground(statusColor).
		Render(statusIcon + " " + statusText)

	// Header layout
	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Overlay0)

	leftPart := title
	rightPart := status

	// Pad middle
	padding := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 4
	if padding < 0 {
		padding = 0
	}
	middle := strings.Repeat(" ", padding)

	return headerStyle.Render(leftPart + middle + rightPart)
}

// renderAgentsPanel renders the agents panel.
func (m Model) renderAgentsPanel(width, height int, t *theme.Theme, ic *icons.IconSet) string {
	focused := m.focus == FocusAgents

	// Panel title
	title := fmt.Sprintf("%s Agents (%d)", ic.Agent, len(m.agents))

	// Content
	var content strings.Builder
	if len(m.agents) == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(t.Subtext).
			Render("No active agents"))
	} else {
		for i, agent := range m.agents {
			selected := i == m.agentCursor && focused
			card := components.NewAgentCard(agent.Info).AsCompact().AsSelected(selected)
			content.WriteString(card.Render())
			if i < len(m.agents)-1 {
				content.WriteString("\n")
			}
		}
	}

	return m.renderPanel(title, content.String(), width, height, focused, t)
}

// renderRequestsPanel renders the pending requests panel.
func (m Model) renderRequestsPanel(width, height int, t *theme.Theme, ic *icons.IconSet) string {
	focused := m.focus == FocusRequests

	// Panel title
	title := fmt.Sprintf("%s Pending Requests (%d)", ic.Pending, len(m.requests))

	// Content
	var content strings.Builder
	if len(m.requests) == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(t.Subtext).
			Render("No pending requests"))
	} else {
		for i, req := range m.requests {
			selected := i == m.requestCursor && focused
			content.WriteString(m.renderRequestItem(req, selected, width-4, t, ic))
			if i < len(m.requests)-1 {
				content.WriteString("\n")
			}
		}
	}

	return m.renderPanel(title, content.String(), width, height, focused, t)
}

// renderRequestItem renders a single request in the list.
func (m Model) renderRequestItem(req RequestData, selected bool, width int, t *theme.Theme, ic *icons.IconSet) string {
	// Tier indicator
	var tierColor lipgloss.Color
	switch strings.ToLower(req.Tier) {
	case "critical":
		tierColor = t.Red
	case "dangerous":
		tierColor = t.Peach
	case "caution":
		tierColor = t.Yellow
	default:
		tierColor = t.Green
	}

	tierBadge := lipgloss.NewStyle().
		Foreground(t.Base).
		Background(tierColor).
		Padding(0, 1).
		Render(strings.ToUpper(req.Tier))

	// Command (truncated)
	maxCmdLen := width - 20
	cmd := req.Command
	if len(cmd) > maxCmdLen && maxCmdLen > 3 {
		cmd = cmd[:maxCmdLen-3] + "..."
	}
	cmdStyle := lipgloss.NewStyle().Foreground(t.Green)

	// Requestor and time
	timeAgo := formatTimeAgo(req.CreatedAt)
	meta := lipgloss.NewStyle().
		Foreground(t.Subtext).
		Render(fmt.Sprintf("by %s • %s", req.RequestorID, timeAgo))

	// Build lines
	line1 := fmt.Sprintf("%s %s", tierBadge, cmdStyle.Render(cmd))
	line2 := meta

	content := line1 + "\n" + line2

	// Selection style
	if selected {
		return lipgloss.NewStyle().
			Background(t.Surface).
			Width(width).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Mauve).
			Render(content)
	}

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay0).
		Render(content)
}

// renderActivityPanel renders the recent activity panel.
func (m Model) renderActivityPanel(width, height int, t *theme.Theme, ic *icons.IconSet) string {
	focused := m.focus == FocusActivity

	// Panel title
	title := "Recent Activity"

	// Content
	var content strings.Builder
	if len(m.activities) == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(t.Subtext).
			Render("No recent activity"))
	} else {
		for i, activity := range m.activities {
			selected := i == m.activityCursor && focused
			content.WriteString(m.renderActivityItem(activity, selected, width-4, t, ic))
			if i < len(m.activities)-1 {
				content.WriteString("\n")
			}
		}
	}

	return m.renderPanel(title, content.String(), width, height, focused, t)
}

// renderActivityItem renders a single activity entry.
func (m Model) renderActivityItem(activity ActivityItem, selected bool, width int, t *theme.Theme, ic *icons.IconSet) string {
	// Action icon
	var actionIcon string
	var actionColor lipgloss.Color
	switch strings.ToLower(activity.Action) {
	case "approved":
		actionIcon = ic.Approved
		actionColor = t.Green
	case "rejected":
		actionIcon = ic.Rejected
		actionColor = t.Red
	case "created":
		actionIcon = ic.Add
		actionColor = t.Blue
	case "executed":
		actionIcon = ic.Success
		actionColor = t.Green
	default:
		actionIcon = ic.Dot
		actionColor = t.Subtext
	}

	actionStyle := lipgloss.NewStyle().Foreground(actionColor)
	timeAgo := formatTimeAgo(activity.Timestamp)

	line := fmt.Sprintf("%s %s %s • %s",
		actionStyle.Render(actionIcon),
		activity.Actor,
		strings.ToLower(activity.Action),
		lipgloss.NewStyle().Foreground(t.Subtext).Render(timeAgo),
	)

	if selected {
		return lipgloss.NewStyle().Background(t.Surface).Render(line)
	}
	return line
}

// renderPanel renders a panel with title and content.
func (m Model) renderPanel(title, content string, width, height int, focused bool, t *theme.Theme) string {
	borderColor := t.Overlay0
	if focused {
		borderColor = t.Mauve
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true)

	panelStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	fullContent := titleStyle.Render(title) + "\n\n" + content

	return panelStyle.Render(fullContent)
}

// renderFooter renders the dashboard footer.
func (m Model) renderFooter(t *theme.Theme, ic *icons.IconSet) string {
	// Stats
	stats := fmt.Sprintf("24h: %d approved │ %d rejected │ avg %.0fs response",
		m.approvedCount24h,
		m.rejectedCount24h,
		m.avgResponseSec,
	)

	// Keybindings
	keybindings := "[tab] switch panel  [↑↓] navigate  [enter] select  [r] refresh  [h] help  [q] quit"

	statsStyle := lipgloss.NewStyle().Foreground(t.Subtext)
	keysStyle := lipgloss.NewStyle().Foreground(t.Overlay1)

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(t.Overlay0).
		Padding(0, 1)

	padding := m.width - lipgloss.Width(stats) - lipgloss.Width(keybindings) - 4
	if padding < 0 {
		padding = 0
	}
	middle := strings.Repeat(" ", padding)

	return footerStyle.Render(statsStyle.Render(stats) + middle + keysStyle.Render(keybindings))
}

// formatTimeAgo formats a time as "X ago" string.
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	return fmt.Sprintf("%dd ago", days)
}

// SetAgents sets the agent data.
func (m *Model) SetAgents(agents []AgentData) {
	m.agents = agents
	if m.agentCursor >= len(agents) {
		m.agentCursor = max(0, len(agents)-1)
	}
}

// SetRequests sets the request data.
func (m *Model) SetRequests(requests []RequestData) {
	m.requests = requests
	if m.requestCursor >= len(requests) {
		m.requestCursor = max(0, len(requests)-1)
	}
}

// SetActivities sets the activity data.
func (m *Model) SetActivities(activities []ActivityItem) {
	m.activities = activities
	if m.activityCursor >= len(activities) {
		m.activityCursor = max(0, len(activities)-1)
	}
}

// SetDaemonStatus sets the daemon connection status.
func (m *Model) SetDaemonStatus(status DaemonStatus) {
	m.daemonStatus = status
}

// SetStats sets the 24h statistics.
func (m *Model) SetStats(approved, rejected int, avgResponse float64) {
	m.approvedCount24h = approved
	m.rejectedCount24h = rejected
	m.avgResponseSec = avgResponse
}

// Run starts the dashboard TUI.
func Run() error {
	m := New()
	// Add sample data for now
	m.daemonStatus = DaemonConnected
	m.SetAgents([]AgentData{
		{Info: components.AgentInfo{Name: "GreenLake", Program: "claude-code", Model: "opus-4.5", Status: components.AgentStatusActive, LastActive: time.Now()}},
		{Info: components.AgentInfo{Name: "BlueDog", Program: "codex-cli", Model: "gpt-5.1", Status: components.AgentStatusActive, LastActive: time.Now().Add(-5 * time.Minute)}},
		{Info: components.AgentInfo{Name: "RedCat", Program: "claude-code", Model: "opus-4.5", Status: components.AgentStatusIdle, LastActive: time.Now().Add(-30 * time.Minute)}},
	})
	m.SetRequests([]RequestData{
		{ID: "req-abc123", Command: "rm -rf ./build", Tier: "dangerous", Status: "pending", RequestorID: "GreenLake", CreatedAt: time.Now().Add(-2 * time.Minute)},
		{ID: "req-def456", Command: "kubectl delete node worker-3", Tier: "critical", Status: "pending", RequestorID: "BlueDog", CreatedAt: time.Now().Add(-5 * time.Minute)},
	})
	m.SetActivities([]ActivityItem{
		{Action: "approved", Actor: "BlueDog", RequestID: "req-xyz", Timestamp: time.Now().Add(-10 * time.Minute)},
		{Action: "created", Actor: "GreenLake", RequestID: "req-abc", Timestamp: time.Now().Add(-15 * time.Minute)},
		{Action: "executed", Actor: "RedCat", RequestID: "req-old", Timestamp: time.Now().Add(-30 * time.Minute)},
	})
	m.SetStats(12, 2, 45)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
