package audio

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/username/sandsysm/internal/system"
)

type Sink struct {
	Name   string
	Volume int
	Muted  bool
	InUse  bool
}

var volumePercentPattern = regexp.MustCompile(`([0-9]+)%`)

func GetSinks() ([]Sink, error) {
	shortOutput, err := system.Run("pactl", "list", "short", "sinks")
	if err != nil {
		return nil, fmt.Errorf("list audio sinks: %w: %s", err, strings.TrimSpace(shortOutput))
	}
	fullOutput, err := system.Run("pactl", "list", "sinks")
	if err != nil {
		return nil, fmt.Errorf("read audio sink details: %w: %s", err, strings.TrimSpace(fullOutput))
	}
	defaultOutput, _ := system.Run("pactl", "get-default-sink")
	defaultSink := strings.TrimSpace(defaultOutput)
	properties := parseSinkDetails(fullOutput)

	sinks := make([]Sink, 0)
	for _, line := range strings.Split(shortOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sink := Sink{Name: fields[1], InUse: fields[1] == defaultSink}
		if detail, ok := properties[sink.Name]; ok {
			sink.Volume = detail.Volume
			sink.Muted = detail.Muted
		}
		sinks = append(sinks, sink)
	}
	return sinks, nil
}

func SetVolume(sink string, volPercent int) error {
	if volPercent < 0 {
		volPercent = 0
	}
	if sink == "" {
		sink = "@DEFAULT_SINK@"
	}
	output, err := system.Run("pactl", "set-sink-volume", sink, fmt.Sprintf("%d%%", volPercent))
	if err != nil {
		return fmt.Errorf("set sink volume: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}

func ToggleMute() error {
	output, err := system.Run("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle")
	if err != nil {
		return fmt.Errorf("toggle sink mute: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}

type sinkDetail struct {
	Volume int
	Muted  bool
}

func parseSinkDetails(output string) map[string]sinkDetail {
	details := make(map[string]sinkDetail)
	var currentName string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Name:") {
			currentName = strings.TrimSpace(strings.TrimPrefix(trimmed, "Name:"))
			details[currentName] = sinkDetail{}
			continue
		}
		if currentName == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Mute:") {
			details[currentName] = sinkDetail{
				Volume: details[currentName].Volume,
				Muted:  strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "Mute:")), "yes"),
			}
			continue
		}
		if strings.HasPrefix(trimmed, "Volume:") {
			match := volumePercentPattern.FindStringSubmatch(trimmed)
			if len(match) != 2 {
				continue
			}
			volume, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			detail := details[currentName]
			detail.Volume = volume
			details[currentName] = detail
		}
	}
	return details
}
