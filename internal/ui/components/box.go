package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	boxUnfocusedStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 1)
	boxFocusedStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("7")).Padding(0, 1)
	boxTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
)

func Box(title, content string, width, height int, focused bool) string {
	if width < 4 {
		width = 4
	}
	if height < 3 {
		height = 3
	}
	style := boxUnfocusedStyle
	if focused {
		style = boxFocusedStyle
	}
	innerWidth := width - 4
	innerHeight := height - 3
	if innerHeight < 0 {
		innerHeight = 0
	}
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		lines[index] = fitBoxLine(line, innerWidth)
	}
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	content = strings.Join(lines, "\n")
	header := boxTitleStyle.Render(" " + title + " ")
	return style.Copy().Width(width - 2).Height(height - 2).Render(header + "\n" + content)
}

func fitBoxLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	runes := []rune(line)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
