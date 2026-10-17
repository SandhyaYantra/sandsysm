package control

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

func GetBrightness() (cur, max int, err error) {
	brightnessPath, err := backlightPath()
	if err != nil {
		return getBrightnessFallback(err)
	}

	currentText, err := system.ReadSysfs(filepath.Join(brightnessPath, "brightness"))
	if err != nil {
		return getBrightnessFallback(err)
	}
	maxText, err := system.ReadSysfs(filepath.Join(brightnessPath, "max_brightness"))
	if err != nil {
		return getBrightnessFallback(err)
	}
	cur, err = strconv.Atoi(currentText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse current brightness: %w", err)
	}
	max, err = strconv.Atoi(maxText)
	if err != nil {
		return 0, 0, fmt.Errorf("parse maximum brightness: %w", err)
	}
	if max <= 0 {
		return getBrightnessFallback(fmt.Errorf("backlight %s has invalid max_brightness", filepath.Base(brightnessPath)))
	}
	return cur, max, nil
}

func SetBrightness(val int) error {
	path, err := backlightPath()
	_, max, maxErr := GetBrightness()
	if maxErr != nil {
		return fmt.Errorf("read display brightness: %w", maxErr)
	}
	if err != nil {
		return setBrightnessFallback(val, max, err)
	}
	if max <= 0 {
		return fmt.Errorf("invalid maximum display brightness %d", max)
	}
	if val < 0 {
		val = 0
	}
	if val > max {
		val = max
	}
	if err = system.WriteSysfs(filepath.Join(path, "brightness"), strconv.Itoa(val)); err == nil {
		log.Printf("display brightness set through %s", filepath.Base(path))
		return nil
	}
	return setBrightnessFallback(val, max, err)
}

func backlightPath() (string, error) {
	paths, err := filepath.Glob("/sys/class/backlight/*")
	if err != nil {
		return "", err
	}
	preferred := make([]string, 0, len(paths))
	other := make([]string, 0, len(paths))
	for _, path := range paths {
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "intel_backlight") || strings.Contains(name, "amdgpu_bl0") {
			preferred = append(preferred, path)
		} else {
			other = append(other, path)
		}
	}
	for _, path := range append(preferred, other...) {
		maxText, err := system.ReadSysfs(filepath.Join(path, "max_brightness"))
		if err != nil {
			continue
		}
		max, err := strconv.Atoi(maxText)
		if err == nil && max > 0 {
			return path, nil
		}
	}
	return "", fmt.Errorf("no usable backlight device with max_brightness > 0")
}

func getBrightnessFallback(sysfsErr error) (int, int, error) {
	currentText, currentErr := system.Run("brightnessctl", "g")
	maxText, maxErr := system.Run("brightnessctl", "m")
	if currentErr == nil && maxErr == nil {
		current, currentParseErr := strconv.Atoi(strings.TrimSpace(currentText))
		maximum, maxParseErr := strconv.Atoi(strings.TrimSpace(maxText))
		if currentParseErr == nil && maxParseErr == nil && maximum > 0 {
			return current, maximum, nil
		}
	}
	return 0, 0, fmt.Errorf("read display brightness via sysfs (%v) and brightnessctl (%v, %v)", sysfsErr, currentErr, maxErr)
}

func setBrightnessFallback(value, maximum int, sysfsErr error) error {
	if maximum <= 0 {
		return fmt.Errorf("invalid maximum brightness %d: %w", maximum, sysfsErr)
	}
	percent := value * 100 / maximum
	percentText := fmt.Sprintf("%d%%", percent)
	brightnessOutput, brightnessErr := system.Run("brightnessctl", "s", percentText)
	if brightnessErr == nil {
		log.Printf("display brightness set through brightnessctl to %s", percentText)
		return nil
	}
	lightOutput, lightErr := system.Run("light", "-S", fmt.Sprintf("%d", percent))
	if lightErr == nil {
		log.Printf("display brightness set through light to %d%%", percent)
		return nil
	}
	return fmt.Errorf("write display brightness via sysfs failed (%w); brightnessctl: %v %s; light: %v %s (need brightnessctl or run with sudo / add udev rule)",
		sysfsErr, brightnessErr, strings.TrimSpace(brightnessOutput), lightErr, strings.TrimSpace(lightOutput))
}
