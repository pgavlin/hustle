// ui/jqinput.go
package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

var (
	jqPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	jqErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// jqResult is sent when the user confirms or cancels the jq input.
type jqResult struct {
	expr      string
	cancelled bool
}

// JQInputModel is a modal text input for entering jq expressions.
type JQInputModel struct {
	input textinput.Model
	err   string
	width int
}

// NewJQInputModel creates a new jq input modal with the given initial expression.
func NewJQInputModel(initialExpr string, width int) JQInputModel {
	ti := textinput.New()
	ti.Placeholder = "jq expression (e.g. select(.level == \"ERROR\"))"
	ti.SetValue(initialExpr)
	ti.Focus()
	return JQInputModel{
		input: ti,
		width: width,
	}
}

// SetError displays an error message below the input.
func (m *JQInputModel) SetError(errMsg string) {
	m.err = errMsg
}

// Value returns the current input text.
func (m JQInputModel) Value() string {
	return m.input.Value()
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
	return m, cmd
}

// View renders the jq input modal.
func (m JQInputModel) View() string {
	prompt := jqPromptStyle.Render("jq filter: ")
	line := prompt + m.input.View()
	if m.err != "" {
		line += "\n" + jqErrorStyle.Render("  error: "+m.err)
	}
	return line
}
