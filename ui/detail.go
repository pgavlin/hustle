package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	logpkg "github.com/pgavlin/hustle/log"
)

type detailMode int

const (
	detailStructured detailMode = iota
	detailRaw
)

var (
	detailLabelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	detailValueStyle  = lipgloss.NewStyle()
	detailHeaderStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("57")).
				Padding(0, 1)
	detailHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// DetailModel displays a single LogRecord in structured or raw JSON format.
type DetailModel struct {
	record logpkg.LogRecord
	mode   detailMode
	width  int
	height int
	scroll int
}

// NewDetailModel creates a DetailModel for the given record.
func NewDetailModel(record logpkg.LogRecord, width, height int) DetailModel {
	return DetailModel{
		record: record,
		mode:   detailStructured,
		width:  width,
		height: height,
	}
}

func (m *DetailModel) toggleMode() {
	if m.mode == detailStructured {
		m.mode = detailRaw
	} else {
		m.mode = detailStructured
	}
	m.scroll = 0
}

// SetSize updates the viewport dimensions.
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles key messages for the detail view.
func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			m.toggleMode()
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		}
	}
	return m, nil
}

// View renders the detail view.
func (m DetailModel) View() tea.View {
	var content string
	switch m.mode {
	case detailStructured:
		content = m.structuredView()
	case detailRaw:
		content = m.rawView()
	}

	header := detailHeaderStyle.Render(m.headerText())
	help := detailHelpStyle.Render("tab: toggle view • esc: back to grid • j/k: scroll")

	lines := strings.Split(content, "\n")
	if m.scroll >= len(lines) {
		m.scroll = max(len(lines)-1, 0)
	}
	visible := lines[m.scroll:]
	bodyHeight := m.height - 3
	if bodyHeight > 0 && len(visible) > bodyHeight {
		visible = visible[:bodyHeight]
	}

	return tea.NewView(header + "\n\n" + strings.Join(visible, "\n") + "\n\n" + help)
}

func (m DetailModel) headerText() string {
	modeLabel := "Structured"
	if m.mode == detailRaw {
		modeLabel = "Raw JSON"
	}
	return fmt.Sprintf(" Log Entry Detail — %s ", modeLabel)
}

func (m DetailModel) structuredView() string {
	var b strings.Builder

	writeField := func(label, value string) {
		b.WriteString(detailLabelStyle.Render(fmt.Sprintf("%12s", label)))
		b.WriteString("  ")
		b.WriteString(detailValueStyle.Render(value))
		b.WriteString("\n")
	}

	writeField("Time", m.record.Time.Format("2006-01-02 15:04:05.000 MST"))
	writeField("Level", m.record.Level)
	writeField("Message", m.record.Msg)

	if len(m.record.Attrs) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("  Attributes"))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 40))
		b.WriteString("\n")

		keys := make([]string, 0, len(m.record.Attrs))
		for k := range m.record.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			writeField(k, formatAttrValue(m.record.Attrs[k]))
		}
	}

	return b.String()
}

func (m DetailModel) rawView() string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(m.record.RawJSON), &raw); err != nil {
		return m.record.RawJSON
	}
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return m.record.RawJSON
	}
	return string(pretty)
}

func formatAttrValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
