package control

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

func GetKbdBrightness() (cur, max int, err error) {
	path, err := keyboardBacklightPath()
	if err != nil {
		return 0, 0, err
	}
	currentText, err := system.ReadSysfs(filepath.Join(path, "brightness"))
	if err != nil {
		return getKeyboardBrightnessFallback(path, err)
	}
	maxText, err := system.ReadSysfs(filepath.Join(path, "max_brightness"))
	if err != nil {
		return getKeyboardBrightnessFallback(path, err)
	}
	cur, err = strconv.Atoi(currentText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse keyboard brightness: %w", err)
	}
	max, err = strconv.Atoi(maxText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse maximum keyboard brightness: %w", err)
	}
	if max <= 0 {
		return getKeyboardBrightnessFallback(path, fmt.Errorf("invalid max_brightness"))
	}
	return cur, max, nil
}

func SetKbdBrightness(value int) error {
	path, pathErr := keyboardBacklightPath()
	_, max, maxErr := GetKbdBrightness()
	if maxErr != nil {
		return fmt.Errorf("read keyboard backlight: %w", maxErr)
	}
	if max <= 0 {
		return fmt.Errorf("invalid maximum keyboard brightness %d", max)
	}
	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}
	if pathErr != nil {
		return setKeyboardBrightnessFallback("", value, max, pathErr)
	}
	if err := system.WriteSysfs(filepath.Join(path, "brightness"), strconv.Itoa(value)); err == nil {
		log.Printf("keyboard brightness set through %s", filepath.Base(path))
		return nil
	} else {
		return setKeyboardBrightnessFallback(filepath.Base(path), value, max, err)
	}
}

func keyboardBacklightPath() (string, error) {
	patterns := []string{
		"/sys/class/leds/*kbd_backlight*",
		"/sys/class/leds/*::kbd_backlight",
		"/sys/class/leds/*::kbd*",
		"/sys/class/leds/tpacpi::kbd_backlight",
		"/sys/class/leds/asus::kbd_backlight",
	}
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true
			maxText, err := system.ReadSysfs(filepath.Join(path, "max_brightness"))
			if err != nil {
				continue
			}
			maximum, err := strconv.Atoi(maxText)
			if err == nil && maximum > 0 {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("keyboard backlight is not available")
}

func getKeyboardBrightnessFallback(path string, sysfsErr error) (int, int, error) {
	device := "*kbd_backlight*"
	if path != "" {
		device = filepath.Base(path)
	}
	currentText, currentErr := system.Run("brightnessctl", "--device", device, "g")
	maxText, maxErr := system.Run("brightnessctl", "--device", device, "m")
	if currentErr == nil && maxErr == nil {
		current, currentParseErr := strconv.Atoi(strings.TrimSpace(currentText))
		maximum, maxParseErr := strconv.Atoi(strings.TrimSpace(maxText))
		if currentParseErr == nil && maxParseErr == nil && maximum > 0 {
			return current, maximum, nil
		}
	}
	return 0, 0, fmt.Errorf("read keyboard brightness via sysfs (%v) and brightnessctl (%v, %v)", sysfsErr, currentErr, maxErr)
}

func setKeyboardBrightnessFallback(device string, value, maximum int, sysfsErr error) error {
	if device == "" {
		device = "*kbd_backlight*"
	}
	percent := value * 100 / maximum
	output, err := system.Run("brightnessctl", "--device", device, "set", fmt.Sprintf("%d%%", percent))
	if err != nil {
		return fmt.Errorf("keyboard backlight sysfs failed (%w); brightnessctl failed: %v %s", sysfsErr, err, strings.TrimSpace(output))
	}
	log.Printf("keyboard brightness set through brightnessctl device %s to %d%%", device, percent)
	return nil
}
