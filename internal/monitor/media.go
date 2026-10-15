package monitor

type MediaStat struct{}

func StartMedia() <-chan MediaStat {
	ch := make(chan MediaStat)
	close(ch)
	return ch
}
