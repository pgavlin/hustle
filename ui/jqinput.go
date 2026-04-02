package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"github.com/pgavlin/hustle/internal/textinput"
	"github.com/pgavlin/hustle/jq"
)

var (
	jqPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	jqErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	jqDocStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Italic(true)
)

// jqResult is sent when the user confirms or cancels the jq input.
type jqResult struct {
	expr      string
	cancelled bool
}

// JQInputModel is a modal text input for entering jq expressions.
type JQInputModel struct {
	input       textinput.Model
	shape       jq.Shape
	suggestions []jq.Suggestion
	err         string
	width       int
}

// NewJQInputModel creates a new jq input modal with the given initial expression.
func NewJQInputModel(initialExpr string, width int, shape jq.Shape) JQInputModel {
	ti := textinput.New()
	ti.Placeholder = "jq expression (e.g. select(.level == \"ERROR\"))"
	ti.SetValue(initialExpr)
	ti.ShowSuggestions = true
	ti.Focus()

	m := JQInputModel{
		input: ti,
		shape: shape,
		width: width,
	}
	m.updateSuggestions()
	return m
}

// SetError displays an error message below the input.
func (m *JQInputModel) SetError(errMsg string) {
	m.err = errMsg
}

// Value returns the current input text.
func (m JQInputModel) Value() string {
	return m.input.Value()
}

func (m *JQInputModel) updateSuggestions() {
	value := m.input.Value()
	pos := m.input.Position()
	m.suggestions = jq.Complete(value, pos, m.shape)
	texts := make([]string, len(m.suggestions))
	for i, s := range m.suggestions {
		texts[i] = s.Text
	}
	m.input.SetSuggestions(texts)
}

// currentDoc returns the doc string for the current suggestion, or "".
func (m JQInputModel) currentDoc() string {
	idx := m.input.CurrentSuggestionIndex()
	if idx < 0 || idx >= len(m.suggestions) {
		return ""
	}
	s := m.suggestions[idx]
	if s.Builtin == "" {
		return ""
	}
	return jq.BuiltinDoc(s.Builtin)
}

// Update handles messages for the jq input modal.
func (m JQInputModel) Update(msg tea.Msg) (JQInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return m, func() tea.Msg {
				return jqResult{expr: m.input.Value()}
			}
		case "esc":
			return m, func() tea.Msg {
				return jqResult{cancelled: true}
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.updateSuggestions()
	return m, cmd
}

// View renders the jq input modal.
func (m JQInputModel) View() string {
	prompt := jqPromptStyle.Render("jq filter: ")
	line := prompt + m.input.View()
	if m.err != "" {
		line += "\n" + jqErrorStyle.Render("  error: "+m.err)
	} else if doc := m.currentDoc(); doc != "" {
		line += "\n" + jqDocStyle.Render("  "+doc)
	}
	return line
}
