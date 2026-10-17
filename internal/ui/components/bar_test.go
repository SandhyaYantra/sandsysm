package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderGaugeWidthStableAcrossPercentDigits(t *testing.T) {
	widthAtNine := lipgloss.Width(RenderGauge("CPU", 9))
	widthAtTen := lipgloss.Width(RenderGauge("CPU", 10))
	if widthAtNine != widthAtTen {
		t.Fatalf("gauge width changed from %d to %d", widthAtNine, widthAtTen)
	}
}

func TestRenderSliderWidthStableAcrossValueDigits(t *testing.T) {
	widthAtNine := lipgloss.Width(RenderSlider(9, 100, ""))
	widthAtTen := lipgloss.Width(RenderSlider(10, 100, ""))
	if widthAtNine != widthAtTen {
		t.Fatalf("slider width changed from %d to %d", widthAtNine, widthAtTen)
	}
}
