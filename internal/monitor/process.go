package monitor

type ProcessStat struct{}

func StartProcess() <-chan ProcessStat {
	ch := make(chan ProcessStat)
	close(ch)
	return ch
}
