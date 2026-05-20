package commands

import "github.com/ArturUshakov/qq-go/internal/command"

func RegisterAll(registry *command.Registry) error {
	registrars := []func(*command.Registry) error{
		RegisterBase,
		RegisterContainers,
		RegisterSystem,
		RegisterCleanup,
	}
	for _, registrar := range registrars {
		if err := registrar(registry); err != nil {
			return err
		}
	}
	return nil
}
