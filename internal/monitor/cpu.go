package monitor

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CpuStat struct {
	Percent float64
}

func StartCpu(ctx context.Context) <-chan CpuStat {
	ch := make(chan CpuStat, 1)
	go func() {
		defer close(ch)
		for {
			percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
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
			if len(percentages) == 0 {
				continue
			}
			select {
			case ch <- CpuStat{Percent: percentages[0]}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}
