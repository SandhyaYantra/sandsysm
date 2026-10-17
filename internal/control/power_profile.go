package control

import (
	"fmt"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

type PowerProfile string

const (
	ProfileBalanced    PowerProfile = "balanced"
	ProfilePowerSaver  PowerProfile = "power-saver"
	ProfilePerformance PowerProfile = "performance"
)

func ListProfiles() ([]PowerProfile, error) {
	output, err := system.Run("powerprofilesctl", "list")
	if err != nil {
		return nil, fmt.Errorf("list power profiles: %w: %s", err, strings.TrimSpace(output))
	}
	available := make(map[PowerProfile]bool)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		for _, profile := range []PowerProfile{ProfileBalanced, ProfilePowerSaver, ProfilePerformance} {
			if strings.HasPrefix(trimmed, string(profile)) || strings.Contains(trimmed, string(profile)) {
				available[profile] = true
			}
		}
	}
	profiles := make([]PowerProfile, 0, len(available))
	for _, profile := range []PowerProfile{ProfileBalanced, ProfilePowerSaver, ProfilePerformance} {
		if available[profile] {
			profiles = append(profiles, profile)
		}
	}
	return profiles, nil
}

func GetActiveProfile() (PowerProfile, error) {
	output, err := system.Run("powerprofilesctl", "get")
	if err != nil {
		return "", fmt.Errorf("get active power profile: %w: %s", err, strings.TrimSpace(output))
	}
	profile := PowerProfile(strings.TrimSpace(output))
	if !validPowerProfile(profile) {
		return "", fmt.Errorf("unknown active power profile %q", profile)
	}
	return profile, nil
}

func SetProfile(profile PowerProfile) error {
	if !validPowerProfile(profile) {
		return fmt.Errorf("unsupported power profile %q", profile)
	}
	output, err := system.Run("powerprofilesctl", "set", string(profile))
	if err != nil {
		return fmt.Errorf("set power profile %q: %w: %s", profile, err, strings.TrimSpace(output))
	}
	return nil
}

func validPowerProfile(profile PowerProfile) bool {
	return profile == ProfileBalanced || profile == ProfilePowerSaver || profile == ProfilePerformance
}
