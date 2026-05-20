package command

import (
	"fmt"
	"strings"

	"github.com/ArturUshakov/qq-go/internal/output"
)

func PrintHelp(registry *Registry) {
	output.Title("qq — утилита для Docker и рабочих команд")
	output.Plain("Использование: qq <команда> [аргументы]")
	for _, group := range registry.Groups() {
		commands := registry.CommandsByGroup(group)
		if len(commands) == 0 {
			continue
		}
		output.Plain("")
		output.Info(registry.GroupDescription(group))
		for _, command := range commands {
			names := strings.Join(command.Names, ", ")
			fmt.Printf("  %-34s %s\n", names, command.Description)
		}
	}
	output.Plain("")
	output.Plain("Примеры:")
	output.Plain("  qq list")
	output.Plain("  qq exec app -r")
	output.Plain("  qq update")
}

func Suggest(registry *Registry, requested string) []string {
	type candidate struct {
		name     string
		distance int
	}
	candidates := make([]candidate, 0)
	for _, name := range registry.Names() {
		distance := levenshtein(requested, name)
		if distance <= 4 || strings.Contains(name, requested) || strings.Contains(requested, name) {
			candidates = append(candidates, candidate{name: name, distance: distance})
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].distance < candidates[i].distance {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	limit := 3
	if len(candidates) < limit {
		limit = len(candidates)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, candidates[i].name)
	}
	return result
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	previous := make([]int, len(b)+1)
	for i := range previous {
		previous[i] = i
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			current[j] = min(current[j-1]+1, previous[j]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(b)]
}

func min(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}
