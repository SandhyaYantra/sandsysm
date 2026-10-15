package monitor

import "context"

type MemStat = RamStat

func StartMem(ctx context.Context) <-chan MemStat {
	return StartRam(ctx)
}
