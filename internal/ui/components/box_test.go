package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBoxRespectsRequestedDimensions(t *testing.T) {
	const width, height = 38, 8
	box := Box("Test", "short\ncontent", width, height, true)
	if got := lipgloss.Width(box); got != width {
		t.Fatalf("Box width = %d, want %d", got, width)
	}
	if got := lipgloss.Height(box); got != height {
		t.Fatalf("Box height = %d, want %d", got, height)
	}
}
