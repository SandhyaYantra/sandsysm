package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	StyleBar   = lipgloss.NewStyle()
	StyleTitle = lipgloss.NewStyle().Bold(true)
)

func RenderBar(used, total uint64, percent float64) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filledCount := int(percent / 10)
	if filledCount > 10 {
		filledCount = 10
	}
	filled := strings.Repeat("█", filledCount)
	empty := strings.Repeat("░", 10-filledCount)
	usedGB := float64(used) / (1024 * 1024 * 1024)
	totalGB := float64(total) / (1024 * 1024 * 1024)
	bar := fmt.Sprintf("[%s%s] %.1f GB / %.1f GB (%.0f%%)", filled, empty, usedGB, totalGB, percent)

	color := lipgloss.Color("#53d769")
	if percent >= 80 {
		color = lipgloss.Color("#ff5d5d")
	} else if percent >= 50 {
		color = lipgloss.Color("#ffd166")
	}
	return StyleBar.Copy().Foreground(color).Render(bar)
}

func RenderGauge(label string, percent float64) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filledCount := int(percent / 10)
	if filledCount > 10 {
		filledCount = 10
	}
	bar := fmt.Sprintf("[%s%s] %.0f%%", strings.Repeat("█", filledCount), strings.Repeat("░", 10-filledCount), percent)
	color := lipgloss.Color("#53d769")
	if percent >= 80 {
		color = lipgloss.Color("#ff5d5d")
	} else if percent >= 50 {
		color = lipgloss.Color("#ffd166")
	}
	if label != "" {
		bar = label + " " + bar
	}
	return StyleBar.Copy().Foreground(color).Render(bar)
}
