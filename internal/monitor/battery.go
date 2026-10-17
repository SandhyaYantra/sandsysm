package monitor

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/username/sandsysm/internal/system"
)

type BatteryStat struct {
	Capacity   int
	Status     string
	PowerNow   float64
	Voltage    float64
	Watts      float64
	IsCharging bool
}

func StartBattery(ctx context.Context) <-chan BatteryStat {
	ch := make(chan BatteryStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stat := readBattery()
				select {
				case ch <- stat:
				case <-ctx.Done():
					return
				default:
					select {
					case <-ch:
					default:
					}
					select {
					case ch <- stat:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch
}

func readBattery() BatteryStat {
	stat := BatteryStat{Capacity: -1, Status: "Unavailable"}
	paths, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil || len(paths) == 0 {
		return stat
	}
	path := paths[0]
	if value, err := system.ReadSysfs(filepath.Join(path, "capacity")); err == nil {
		if capacity, err := strconv.Atoi(value); err == nil {
			stat.Capacity = capacity
		}
	}
	if value, err := system.ReadSysfs(filepath.Join(path, "status")); err == nil {
		stat.Status = value
		stat.IsCharging = strings.EqualFold(value, "Charging")
	}
	if value, err := system.ReadSysfs(filepath.Join(path, "voltage_now")); err == nil {
		if voltage, err := strconv.ParseFloat(value, 64); err == nil {
			stat.Voltage = voltage / 1e6
		}
	}
	if value, err := system.ReadSysfs(filepath.Join(path, "power_now")); err == nil {
		if power, err := strconv.ParseFloat(value, 64); err == nil {
			stat.PowerNow = power / 1e6
			stat.Watts = stat.PowerNow
			return stat
		}
	}
	currentText, currentErr := system.ReadSysfs(filepath.Join(path, "current_now"))
	if currentErr == nil && stat.Voltage > 0 {
		if current, err := strconv.ParseFloat(currentText, 64); err == nil {
			stat.Watts = current * stat.Voltage / 1e6
			stat.PowerNow = stat.Watts
		}
	}
	return stat
}
