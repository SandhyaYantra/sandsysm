package monitor

import (
	"context"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessStat struct {
	PID  int32
	Name string
	CPU  float64
	Mem  float64
}

func StartProcess(ctx context.Context) <-chan []ProcessStat {
	ch := make(chan []ProcessStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processes, err := process.Processes()
				if err != nil {
					continue
				}

				stats := make([]ProcessStat, 0, len(processes))
				for _, proc := range processes {
					name, err := proc.Name()
					if err != nil {
						continue
					}
					cpuPercent, err := proc.CPUPercent()
					if err != nil {
						continue
					}
					memPercent, err := proc.MemoryPercent()
					if err != nil {
						memPercent = 0
					}
					stats = append(stats, ProcessStat{
						PID:  proc.Pid,
						Name: name,
						CPU:  cpuPercent,
						Mem:  float64(memPercent),
					})
				}
				sort.Slice(stats, func(i, j int) bool {
					return stats[i].CPU > stats[j].CPU
				})
				if len(stats) > 10 {
					stats = stats[:10]
				}

				select {
				case ch <- stats:
				case <-ctx.Done():
					return
				default:
					select {
					case <-ch:
					default:
					}
					select {
					case ch <- stats:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch
}
