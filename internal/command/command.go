package command

import (
	"fmt"
	"sort"
	"strings"
)

type Handler func(args []string) error

type Command struct {
	Names       []string
	Group       string
	Description string
	Usage       string
	Run         Handler
}

type Registry struct {
	commands map[string]Command
	groups   map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
		groups: map[string]string{
			"base":      "Основные команды",
			"container": "Команды контейнеров",
			"system":    "Системные команды",
			"cleanup":   "Очистка Docker",
		},
	}
}

func (registry *Registry) Register(command Command) error {
	if len(command.Names) == 0 {
		return fmt.Errorf("у команды нет имени")
	}
	if command.Group == "" {
		command.Group = "base"
	}
	for _, name := range command.Names {
		if _, exists := registry.commands[name]; exists {
			return fmt.Errorf("команда %q уже зарегистрирована", name)
		}
		registry.commands[name] = command
	}
	return nil
}

func (registry *Registry) Get(name string) (Command, bool) {
	command, exists := registry.commands[name]
	return command, exists
}

func (registry *Registry) Names() []string {
	unique := make(map[string]struct{})
	for _, cmd := range registry.commands {
		unique[cmd.Names[0]] = struct{}{}
	}
	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (registry *Registry) AllNames() []string {
	names := make([]string, 0, len(registry.commands))
	for name := range registry.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (registry *Registry) CommandsByGroup(group string) []Command {
	seen := make(map[string]struct{})
	commands := make([]Command, 0)
	for _, command := range registry.commands {
		if command.Group != group {
			continue
		}
		key := strings.Join(command.Names, "|")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		commands = append(commands, command)
	}
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Names[0] < commands[j].Names[0]
	})
	return commands
}

func (registry *Registry) Groups() []string {
	groups := []string{"base", "container", "system", "cleanup"}
	return groups
}

func (registry *Registry) GroupDescription(group string) string {
	return registry.groups[group]
}
