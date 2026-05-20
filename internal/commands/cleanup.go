package commands

import (
	"fmt"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/execx"
	"github.com/ArturUshakov/qq-go/internal/output"
)

func RegisterCleanup(registry *command.Registry) error {
	commands := []command.Command{
		{Names: []string{"cleanup-docker-images", "-dni"}, Group: "cleanup", Description: "Удалить dangling Docker images", Run: cleanupDockerImagesCommand},
		{Names: []string{"prune-builder", "-pb"}, Group: "cleanup", Description: "Удалить неиспользуемый Docker builder cache", Run: pruneBuilderCommand},
	}
	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func cleanupDockerImagesCommand(args []string) error {
	return cleanupDockerImages(false, false, true)
}

func pruneBuilderCommand(args []string) error {
	return pruneBuilder(false, true)
}

func cleanupDockerImages(dryRun bool, safe bool, verbose bool) error {
	output.Info("Поиск dangling images...")
	result, err := execx.Output("docker", "images", "-f", "dangling=true", "-q")
	if err != nil {
		return err
	}
	imageIDs := splitLines(result.Stdout)
	if len(imageIDs) == 0 {
		output.Success("Нет dangling images для удаления")
		return nil
	}
	if dryRun {
		output.Info("[dry-run] Найдено images для удаления: %d", len(imageIDs))
		for _, imageID := range imageIDs {
			output.Plain("  %s", imageID)
		}
		return nil
	}
	idsToRemove := imageIDs
	if safe {
		usedResult, err := execx.Output("docker", "ps", "-a", "--format", "{{.Image}}")
		if err != nil {
			return err
		}
		used := make(map[string]struct{})
		for _, image := range splitLines(usedResult.Stdout) {
			used[image] = struct{}{}
		}
		idsToRemove = make([]string, 0)
		for _, imageID := range imageIDs {
			if _, exists := used[imageID]; !exists {
				idsToRemove = append(idsToRemove, imageID)
			}
		}
		if len(idsToRemove) == 0 {
			output.Warn("Все найденные images используются. Удаление пропущено")
			return nil
		}
	}
	args := append([]string{"rmi"}, idsToRemove...)
	var removeErr error
	if verbose {
		removeErr = execx.RunPassthrough("docker", args...)
	} else {
		removeErr = execx.RunQuiet("docker", args...)
	}
	if removeErr != nil {
		return fmt.Errorf("ошибка при удалении images: %w", removeErr)
	}
	output.Success("Удалено dangling images: %d", len(idsToRemove))
	return nil
}

func pruneBuilder(dryRun bool, verbose bool) error {
	output.Info("Очистка builder cache...")
	if dryRun {
		output.Info("[dry-run] Был бы выполнен: docker builder prune -f")
		return nil
	}
	if verbose {
		if err := execx.RunPassthrough("docker", "builder", "prune", "-f"); err != nil {
			return err
		}
	} else if err := execx.RunQuiet("docker", "builder", "prune", "-f"); err != nil {
		return err
	}
	output.Success("Builder cache очищен")
	return nil
}
