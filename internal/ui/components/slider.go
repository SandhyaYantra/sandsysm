package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderSlider(cur, max int, label string) string {
	if max < 0 {
		max = 0
	}
	if cur < 0 {
		cur = 0
	}
	if cur > max {
		cur = max
	}
	percent := 0
	if max > 0 {
		percent = cur * 100 / max
	}
	filledCount := percent / 10
	if filledCount > 10 {
		filledCount = 10
	}
	bar := fmt.Sprintf("[%s%s] %3d/%3d", strings.Repeat("█", filledCount), strings.Repeat("░", 10-filledCount), cur, max)
	color := lipgloss.Color("#53d769")
	if percent >= 80 {
		color = lipgloss.Color("#ff5d5d")
	} else if percent >= 50 {
		color = lipgloss.Color("#ffd166")
	}
	if label != "" {
		return label + " " + StyleBar.Copy().Foreground(color).Render(bar)
	}
	return StyleBar.Copy().Foreground(color).Render(bar)
}
