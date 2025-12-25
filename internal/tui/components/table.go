// Package components provides table components.
package components

import (
	"strings"

	"github.com/Dicklesworthstone/slb/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Column defines a table column.
type Column struct {
	Header   string
	Width    int // Fixed width (0 = auto)
	MinWidth int
	MaxWidth int
	Align    lipgloss.Position
	Sortable bool
	SortKey  string // Key for sorting
}

// Table renders data in a styled table.
type Table struct {
	Columns     []Column
	Rows        [][]string
	SelectedRow int
	ShowHeader  bool
	Striped     bool
	Compact     bool
	BorderStyle lipgloss.Border
	MaxWidth    int
}

// NewTable creates a new table component.
func NewTable(columns []Column) *Table {
	return &Table{
		Columns:     columns,
		ShowHeader:  true,
		Striped:     true,
		SelectedRow: -1,
		BorderStyle: lipgloss.NormalBorder(),
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cells ...string) *Table {
	t.Rows = append(t.Rows, cells)
	return t
}

// WithRows sets all rows.
func (t *Table) WithRows(rows [][]string) *Table {
	t.Rows = rows
	return t
}

// WithSelection sets the selected row index.
func (t *Table) WithSelection(idx int) *Table {
	t.SelectedRow = idx
	return t
}

// AsCompact sets compact mode.
func (t *Table) AsCompact() *Table {
	t.Compact = true
	return t
}

// WithoutStripes disables striping.
func (t *Table) WithoutStripes() *Table {
	t.Striped = false
	return t
}

// WithMaxWidth sets the maximum table width.
func (t *Table) WithMaxWidth(width int) *Table {
	t.MaxWidth = width
	return t
}

// Render renders the table.
func (t *Table) Render() string {
	th := theme.Current

	if len(t.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	widths := t.calculateWidths()

	var lines []string

	// Header
	if t.ShowHeader {
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(th.Blue).
			Background(th.Surface)

		var headerCells []string
		for i, col := range t.Columns {
			cell := t.padCell(col.Header, widths[i], col.Align)
			headerCells = append(headerCells, headerStyle.Render(cell))
		}
		lines = append(lines, strings.Join(headerCells, " "))

		// Separator
		sepStyle := lipgloss.NewStyle().Foreground(th.Overlay0)
		sep := sepStyle.Render(strings.Repeat("─", t.totalWidth(widths)))
		lines = append(lines, sep)
	}

	// Rows
	for rowIdx, row := range t.Rows {
		var cells []string

		// Determine row style
		baseStyle := lipgloss.NewStyle().Foreground(th.Text)

		if rowIdx == t.SelectedRow {
			baseStyle = baseStyle.Background(th.Surface1).Bold(true)
		} else if t.Striped && rowIdx%2 == 1 {
			baseStyle = baseStyle.Background(th.Surface0)
		}

		for i, col := range t.Columns {
			cellContent := ""
			if i < len(row) {
				cellContent = row[i]
			}
			cell := t.padCell(cellContent, widths[i], col.Align)
			cells = append(cells, baseStyle.Render(cell))
		}
		lines = append(lines, strings.Join(cells, " "))
	}

	return strings.Join(lines, "\n")
}

// calculateWidths calculates column widths.
func (t *Table) calculateWidths() []int {
	widths := make([]int, len(t.Columns))

	// Start with header widths or fixed widths
	for i, col := range t.Columns {
		if col.Width > 0 {
			widths[i] = col.Width
		} else {
			widths[i] = len(col.Header)
		}

		if col.MinWidth > 0 && widths[i] < col.MinWidth {
			widths[i] = col.MinWidth
		}
	}

	// Check data widths
	for _, row := range t.Rows {
		for i, cell := range row {
			if i >= len(widths) {
				continue
			}
			if t.Columns[i].Width == 0 { // Only auto-size columns
				cellWidth := len(cell)
				if cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			}
		}
	}

	// Apply max widths
	for i, col := range t.Columns {
		if col.MaxWidth > 0 && widths[i] > col.MaxWidth {
			widths[i] = col.MaxWidth
		}
	}

	return widths
}

// totalWidth calculates the total table width.
func (t *Table) totalWidth(widths []int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	// Add separators between columns
	if len(widths) > 1 {
		total += len(widths) - 1
	}
	return total
}

// padCell pads a cell to the specified width with alignment.
func (t *Table) padCell(content string, width int, align lipgloss.Position) string {
	if len(content) > width {
		// Truncate with ellipsis
		if width > 3 {
			return content[:width-3] + "..."
		}
		return content[:width]
	}

	padding := width - len(content)
	switch align {
	case lipgloss.Right:
		return strings.Repeat(" ", padding) + content
	case lipgloss.Center:
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + content + strings.Repeat(" ", rightPad)
	default: // Left
		return content + strings.Repeat(" ", padding)
	}
}

// RenderTable is a convenience function.
func RenderTable(columns []Column, rows [][]string) string {
	return NewTable(columns).WithRows(rows).Render()
}
