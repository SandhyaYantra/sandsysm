package audio

import (
	"fmt"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

type Source struct {
	Name  string
	Desc  string
	InUse bool
}

func GetSources() ([]Source, error) {
	shortOutput, err := system.Run("pactl", "list", "short", "sources")
	if err != nil {
		return nil, fmt.Errorf("list audio sources: %w: %s", err, strings.TrimSpace(shortOutput))
	}
	fullOutput, err := system.Run("pactl", "list", "sources")
	if err != nil {
		return nil, fmt.Errorf("read audio source details: %w: %s", err, strings.TrimSpace(fullOutput))
	}
	defaultOutput, _ := system.Run("pactl", "get-default-source")
	defaultSource := strings.TrimSpace(defaultOutput)
	descriptions := parseSourceDescriptions(fullOutput)

	sources := make([]Source, 0)
	for _, line := range strings.Split(shortOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sources = append(sources, Source{
			Name:  fields[1],
			Desc:  descriptions[fields[1]],
			InUse: fields[1] == defaultSource,
		})
	}
	return sources, nil
}

func SetDefaultSource(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("source name cannot be empty")
	}
	output, err := system.Run("pactl", "set-default-source", name)
	if err != nil {
		return fmt.Errorf("set default audio source: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}

func parseSourceDescriptions(output string) map[string]string {
	descriptions := make(map[string]string)
	var name string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "Name:"))
		}
		if name != "" && strings.HasPrefix(trimmed, "Description:") {
			descriptions[name] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Description:"))
		}
	}
	return descriptions
}
