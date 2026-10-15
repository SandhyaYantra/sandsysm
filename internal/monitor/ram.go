package monitor

import (
	"context"
	"time"

	"github.com/username/sandsysm/internal/system"
)

type RamStat struct {
	Free      uint64
	Used      uint64
	Cache     uint64
	Available uint64
	Total     uint64
	Buffers   uint64
}

func StartRam(ctx context.Context) <-chan RamStat {
	ch := make(chan RamStat, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				memInfo, err := system.ReadMemInfo()
				if err != nil {
					continue
				}

				toMB := func(value uint64) uint64 { return value / 1024 }
				totalKB := memInfo["MemTotal"]
				freeKB := memInfo["MemFree"]
				buffersKB := memInfo["Buffers"]
				cacheKB := memInfo["Cached"]
				usedKB := totalKB
				for _, reclaimableKB := range []uint64{freeKB, buffersKB, cacheKB} {
					if reclaimableKB >= usedKB {
						usedKB = 0
						break
					}
					usedKB -= reclaimableKB
				}

				stat := RamStat{
					Free:      toMB(freeKB),
					Used:      toMB(usedKB),
					Cache:     toMB(cacheKB),
					Available: toMB(memInfo["MemAvailable"]),
					Total:     toMB(totalKB),
					Buffers:   toMB(buffersKB),
				}
				select {
				case ch <- stat:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}
