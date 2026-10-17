package control

import "github.com/username/sandsysm/internal/control/audio"

type Sink = audio.Sink

func GetSinks() ([]Sink, error) {
	return audio.GetSinks()
}

func SetVolume(sink string, volPercent int) error {
	return audio.SetVolume(sink, volPercent)
}

func ToggleMute() error {
	return audio.ToggleMute()
}
