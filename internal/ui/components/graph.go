package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var graphBlocks = [...]string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

func RenderGraph(history []float64, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(history) > width {
		history = history[len(history)-width:]
	}
	values := make([]float64, width)
	copy(values[width-len(history):], history)

	maximum := 0.0
	for _, value := range values {
		if value > maximum && !math.IsNaN(value) && !math.IsInf(value, 0) {
			maximum = value
		}
	}
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		var line strings.Builder
		for _, value := range values {
			level := 0.0
			if maximum > 0 && value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
				level = value / maximum * float64(height*8)
			}
			cellLevel := level - float64(height-row-1)*8
			blockIndex := int(math.Ceil(cellLevel))
			if blockIndex < 0 {
				blockIndex = 0
			}
			if blockIndex > 8 {
				blockIndex = 8
			}
			color := graphColor(level / float64(height*8))
			line.WriteString(StyleBar.Copy().Foreground(color).Render(graphBlocks[blockIndex]))
		}
		rows[row] = line.String()
	}
	return strings.Join(rows, "\n")
}

func graphColor(intensity float64) lipgloss.Color {
	if intensity >= 0.8 {
		return lipgloss.Color("#ff5d5d")
	}
	if intensity >= 0.5 {
		return lipgloss.Color("#ffd166")
	}
	return lipgloss.Color("#53d769")
}
