package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	helpTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	helpKeyStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Width(16)
	helpDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).MarginTop(1)
	helpBorderStyle  = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("245")).
				Padding(0, 1)
)

func renderHelpOverlay() string {
	binding := func(key, desc string) string {
		return helpKeyStyle.Render(key) + helpDescStyle.Render(desc)
	}

	var lines []string
	lines = append(lines, helpTitleStyle.Render("hustle -- Key Bindings"))
	lines = append(lines, "")

	lines = append(lines, helpSectionStyle.Render("Navigation"))
	lines = append(lines, binding("j / down", "Move down"))
	lines = append(lines, binding("k / up", "Move up"))
	lines = append(lines, binding("g", "Go to top"))
	lines = append(lines, binding("G", "Go to bottom"))
	lines = append(lines, binding("enter", "View record details"))
	lines = append(lines, binding("esc", "Back / dismiss"))

	lines = append(lines, helpSectionStyle.Render("Filtering"))
	lines = append(lines, binding("/", "Quick filter"))
	lines = append(lines, binding("j", "jq filter expression"))
	lines = append(lines, binding("tab", "Cycle completions"))
	lines = append(lines, binding("shift+tab", "Cycle back"))

	lines = append(lines, helpSectionStyle.Render("General"))
	lines = append(lines, binding("?", "Toggle this help"))
	lines = append(lines, binding("q / ctrl+c", "Quit"))

	return strings.Join(lines, "\n")
}

// placeOverlay renders a bordered dialog centered over the background.
func placeOverlay(width, height int, content, background string) string {
	// Truncate content if too tall.
	maxH := height * 3 / 4
	lines := strings.Split(content, "\n")
	if len(lines) > maxH-2 {
		lines = lines[:maxH-2]
	}
	content = strings.Join(lines, "\n")

	// Measure content width.
	contentWidth := 0
	for _, line := range lines {
		w := ansi.StringWidth(line)
		if w > contentWidth {
			contentWidth = w
		}
	}

	dialogWidth := contentWidth + 4
	if dialogWidth > width-2 {
		dialogWidth = width - 2
	}

	dialog := helpBorderStyle.Width(dialogWidth).Render(content)

	// Split background and dialog into lines.
	bgLines := strings.Split(background, "\n")
	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	dialogLines := strings.Split(dialog, "\n")
	dh := len(dialogLines)
	dw := 0
	for _, dl := range dialogLines {
		w := ansi.StringWidth(dl)
		if w > dw {
			dw = w
		}
	}

	// Center the dialog.
	startY := (height - dh) / 2
	startX := (width - dw) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, dl := range dialogLines {
		y := startY + i
		if y >= len(bgLines) {
			break
		}
		bgLine := bgLines[y]
		dlWidth := ansi.StringWidth(dl)

		left := ansi.Truncate(bgLine, startX, "")
		leftWidth := ansi.StringWidth(left)
		if leftWidth < startX {
			left += strings.Repeat(" ", startX-leftWidth)
		}

		rightStart := startX + dlWidth
		right := ""
		bgWidth := ansi.StringWidth(bgLine)
		if rightStart < bgWidth {
			right = ansi.TruncateLeft(bgLine, rightStart, "")
		}

		bgLines[y] = left + dl + right
	}

	return strings.Join(bgLines, "\n")
}
