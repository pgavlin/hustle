// ui/app.go
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgavlin/jq"
	"github.com/pgavlin/tea-grid/grid"

	filterpkg "github.com/pgavlin/hustle/filter"
	logpkg "github.com/pgavlin/hustle/log"
)

type appView int

const (
	viewGrid appView = iota
	viewDetail
	viewJQInput
)

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("252")).
	Background(lipgloss.Color("235")).
	Padding(0, 1)

// Model is the top-level bubbletea model for hustle.
type Model struct {
	records    []logpkg.LogRecord
	skipped    int
	formatName string
	inputShape jq.Shape
	grid       grid.Model[logpkg.LogRecord]
	detail     DetailModel
	jqInput    JQInputModel
	jqFilter   *filterpkg.JQFilter
	jqExpr     string
	view       appView
	showHelp   bool
	gridReady  bool
	width      int
	height     int
}

// New creates the top-level model with loaded records.
func New(records []logpkg.LogRecord, skipped int, formatName string) Model {
	return Model{
		records:    records,
		skipped:    skipped,
		formatName: formatName,
		inputShape: jq.InferShape(recordsToMaps(records)),
		view:       viewGrid,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) rebuildGrid() {
	var extFilter func(logpkg.LogRecord) bool
	if m.jqFilter != nil {
		extFilter = m.jqFilter.Match
	}
	m.grid = newLogGrid(m.records, m.width, m.gridHeight(), extFilter)
}

func (m *Model) resizeGrid() {
	m.grid.SetWidth(m.width)
	m.grid.SetHeight(m.gridHeight())
}

func (m *Model) gridHeight() int {
	reserved := 1 // status bar
	if m.view == viewJQInput {
		reserved = 2 // input + error line
	}
	return m.height - reserved
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.gridReady {
			m.rebuildGrid()
			m.gridReady = true
		} else {
			m.resizeGrid()
		}
		m.detail.SetSize(m.width, m.height)
		return m, nil

	case jqResult:
		if msg.cancelled {
			m.view = viewGrid
			m.resizeGrid()
			return m, nil
		}
		expr := msg.expr
		if expr == "" {
			m.jqFilter = nil
			m.jqExpr = ""
			m.view = viewGrid
			m.rebuildGrid()
			return m, nil
		}
		f, err := filterpkg.Compile(expr)
		if err != nil {
			m.jqInput.SetError(err.Error())
			return m, nil
		}
		m.jqFilter = f
		m.jqExpr = expr
		m.view = viewGrid
		m.rebuildGrid()
		return m, nil

	case tea.KeyPressMsg:
		// Help overlay consumes all keys except dismiss keys.
		if m.showHelp {
			switch msg.String() {
			case "esc", "?", "q":
				m.showHelp = false
			}
			return m, nil
		}

		if m.view == viewGrid && m.grid.Filtering() {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == viewGrid {
				return m, tea.Quit
			}
		case "?":
			if m.view == viewGrid {
				m.showHelp = true
				return m, nil
			}
		case "enter":
			if m.view == viewGrid {
				if rec, ok := m.grid.FocusedRowData(); ok {
					m.detail = NewDetailModel(rec, m.width, m.height)
					m.view = viewDetail
					return m, nil
				}
			}
		case "esc":
			if m.view == viewDetail {
				m.view = viewGrid
				return m, nil
			}
		case ":":
			if m.view == viewGrid {
				m.jqInput = NewJQInputModel(m.jqExpr, m.width, m.inputShape)
				m.view = viewJQInput
				m.resizeGrid()
				return m, nil
			}
		}
	}

	switch m.view {
	case viewGrid:
		var cmd tea.Cmd
		m.grid, cmd = m.grid.Update(msg)
		return m, cmd
	case viewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	case viewJQInput:
		var cmd tea.Cmd
		m.jqInput, cmd = m.jqInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var v tea.View
	switch m.view {
	case viewDetail:
		v = m.detail.View()
	case viewJQInput:
		v = tea.NewView(m.grid.View() + "\n" + m.jqInput.View())
	default:
		base := m.grid.View() + "\n" + m.statusBar()
		if m.showHelp {
			base = placeOverlay(m.width, m.height, renderHelpOverlay(), base)
		}
		v = tea.NewView(base)
	}
	v.AltScreen = true
	return v
}

func (m Model) statusBar() string {
	text := fmt.Sprintf(" [%s] %d records", m.formatName, len(m.records))
	if m.skipped > 0 {
		text += fmt.Sprintf(", %d skipped", m.skipped)
	}
	if m.jqExpr != "" {
		text += fmt.Sprintf(" | jq: %s", m.jqExpr)
	}

	hint := "? help"
	padding := m.width - len(text) - len(hint) - 2
	if padding > 0 {
		text += strings.Repeat(" ", padding) + hint
	}

	return statusBarStyle.Width(m.width).Render(text)
}

// recordsToMaps returns an iterator that yields each LogRecord as a map[string]any
// for jq shape inference.
func recordsToMaps(records []logpkg.LogRecord) func(yield func(map[string]any) bool) {
	return func(yield func(map[string]any) bool) {
		for _, rec := range records {
			m := make(map[string]any, 3+len(rec.Attrs))
			m["time"] = rec.Time.Format(time.RFC3339Nano)
			m["level"] = rec.Level
			m["msg"] = rec.Msg
			for _, kv := range rec.Attrs {
				m[kv.Key] = kv.Value
			}
			if !yield(m) {
				return
			}
		}
	}
}
