package audio

import (
	"fmt"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

func ToggleMicMute() error {
	output, err := system.Run("pactl", "set-source-mute", "@DEFAULT_SOURCE@", "toggle")
	if err != nil {
		return fmt.Errorf("toggle microphone mute: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}
