package monitor

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

type DiskStat struct {
	Mount   string
	Used    uint64
	Free    uint64
	Total   uint64
	Percent float64
}

func StartDisk(ctx context.Context) <-chan []DiskStat {
	ch := make(chan []DiskStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				partitions, err := disk.Partitions(true)
				if err != nil {
					continue
				}

				stats := make([]DiskStat, 0, len(partitions))
				for _, partition := range partitions {
					if partition.Fstype == "tmpfs" || partition.Fstype == "devtmpfs" {
						continue
					}
					usage, err := disk.Usage(partition.Mountpoint)
					if err != nil {
						continue
					}
					stats = append(stats, DiskStat{
						Mount:   partition.Mountpoint,
						Used:    usage.Used / (1024 * 1024 * 1024),
						Free:    usage.Free / (1024 * 1024 * 1024),
						Total:   usage.Total / (1024 * 1024 * 1024),
						Percent: usage.UsedPercent,
					})
				}
				select {
				case ch <- stats:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
