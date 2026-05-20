package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/execx"
	"github.com/ArturUshakov/qq-go/internal/output"
)

func RegisterCleanup(registry *command.Registry) error {
	commands := []command.Command{
		{Names: []string{"cleanup-docker-images", "-dni"}, Group: "cleanup", Description: "Удалить dangling Docker images", Run: cleanupDockerImagesCommand},
		{Names: []string{"prune-builder", "-pb"}, Group: "cleanup", Description: "Удалить неиспользуемый Docker builder cache", Run: pruneBuilderCommand},
		{Names: []string{"clear", "-clr"}, Group: "cleanup", Description: "Очистить Docker: dangling images, builder cache, volumes", Run: clearDockerCommand},
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

func clearDockerCommand(args []string) error {
	flags := parseFlags(args)
	if flags["-h"] || flags["--help"] {
		printClearHelp()
		return nil
	}
	dryRun := flags["--dry-run"]
	safe := flags["--safe"]
	verbose := flags["--verbose"]
	skipConfirm := flags["--yes"] || flags["--force"]
	if !dryRun && !skipConfirm {
		fmt.Print("Уверены, что хотите продолжить очистку Docker? (y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
			output.Warn("Очистка отменена пользователем")
			return nil
		}
	}
	output.Info("Объём Docker перед очисткой:")
	_ = execx.RunPassthrough("docker", "system", "df")
	if err := cleanupDockerImages(dryRun, safe, verbose); err != nil {
		return err
	}
	if err := pruneBuilder(dryRun, verbose); err != nil {
		return err
	}
	if err := cleanupVolumes(dryRun, verbose); err != nil {
		return err
	}
	output.Info("\nОбъём Docker после очистки:")
	_ = execx.RunPassthrough("docker", "system", "df")
	return nil
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

func cleanupVolumes(dryRun bool, verbose bool) error {
	output.Info("Очистка неиспользуемых volumes...")
	if dryRun {
		output.Info("[dry-run] Был бы выполнен: docker volume prune -f")
		return nil
	}
	if verbose {
		if err := execx.RunPassthrough("docker", "volume", "prune", "-f"); err != nil {
			return err
		}
	} else if err := execx.RunQuiet("docker", "volume", "prune", "-f"); err != nil {
		return err
	}
	output.Success("Неиспользуемые volumes удалены")
	return nil
}

func parseFlags(args []string) map[string]bool {
	flags := make(map[string]bool)
	for _, arg := range args {
		flags[arg] = true
	}
	return flags
}

func printClearHelp() {
	output.Plain("Очистка Docker-ресурсов")
	output.Plain("Использование: qq clear [флаги]")
	output.Plain("Флаги:")
	output.Plain("  --dry-run   Показать действия без удаления")
	output.Plain("  --safe      Удалять только явно неиспользуемые images")
	output.Plain("  --verbose   Показывать вывод Docker-команд")
	output.Plain("  --yes       Пропустить подтверждение")
	output.Plain("  --force     То же, что --yes")
}
