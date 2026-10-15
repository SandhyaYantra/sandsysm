package ui

type App struct{}

func NewApp() *App {
	return &App{}
}

func (a *App) Run() error {
	return nil
}
