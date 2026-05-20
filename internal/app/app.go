package app

import (
	"fmt"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/commands"
	"github.com/ArturUshakov/qq-go/internal/output"
)

type App struct {
	registry *command.Registry
}

func New() (*App, error) {
	registry := command.NewRegistry()
	if err := commands.RegisterAll(registry); err != nil {
		return nil, err
	}
	return &App{registry: registry}, nil
}

func (app *App) Run(args []string) int {
	if len(args) == 0 {
		command.PrintHelp(app.registry)
		return 0
	}
	name := args[0]
	cmd, exists := app.registry.Get(name)
	if !exists {
		output.Error("Неизвестная команда: %s", name)
		if suggestions := command.Suggest(app.registry, name); len(suggestions) > 0 {
			output.Info("Похожие команды:")
			for _, suggestion := range suggestions {
				output.Plain("  %s", suggestion)
			}
		}
		output.Plain("Используйте: qq help")
		return 1
	}
	if err := cmd.Run(args[1:]); err != nil {
		output.Error("%s", err.Error())
		return 1
	}
	return 0
}

func (app *App) Registry() *command.Registry {
	return app.registry
}

func Register(registry *command.Registry, cmd command.Command) error {
	if err := registry.Register(cmd); err != nil {
		return fmt.Errorf("register %s: %w", cmd.Names[0], err)
	}
	return nil
}
