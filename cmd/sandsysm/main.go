package main

import (
	"github.com/username/sandsysm/internal/config"
	"github.com/username/sandsysm/internal/ui"
)

func main() {
	config.Load()
	if err := ui.NewApp().Run(); err != nil {
		panic(err)
	}
}
