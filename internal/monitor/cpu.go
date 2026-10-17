package monitor

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CpuStat struct {
	Total   float64
	PerCore []float64
}

func StartCpu(ctx context.Context) <-chan CpuStat {
	ch := make(chan CpuStat, 1)
	go func() {
		defer close(ch)
		for {
			corePercentages, err := cpu.PercentWithContext(ctx, time.Second, true)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				timer := time.NewTimer(time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				continue
			}
			if len(corePercentages) == 0 {
				continue
			}
			total := 0.0
			for _, percentage := range corePercentages {
				total += percentage
			}
			select {
			case ch <- CpuStat{Total: total / float64(len(corePercentages)), PerCore: corePercentages}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
