package monitor

import (
	"context"
	"path/filepath"
	"strconv"
	"time"

	"github.com/username/sandsysm/internal/system"
)

type GpuStat struct {
	Percent float64
}

func StartGpu(ctx context.Context) <-chan GpuStat {
	ch := make(chan GpuStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				percent := float64(0)
				paths, err := filepath.Glob("/sys/class/drm/card*/device/gpu_busy_percent")
				if err == nil {
					for _, path := range paths {
						value, err := system.ReadSysfs(path)
						if err != nil {
							continue
						}
						percent, err = strconv.ParseFloat(value, 64)
						if err == nil {
							break
						}
						percent = 0
					}
				}
				select {
				case ch <- GpuStat{Percent: percent}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
