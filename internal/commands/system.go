package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/SolasWyrd/qq-go/internal/command"
	"github.com/SolasWyrd/qq-go/internal/execx"
	"github.com/SolasWyrd/qq-go/internal/output"
)

func RegisterSystem(registry *command.Registry) error {
	commands := []command.Command{
		{Names: []string{"chmod", "-ch"}, Group: "system", Description: "Рекурсивно выставить chmod 775 для текущей директории", Run: chmodAll},
		{Names: []string{"generate-password-hash", "-gph"}, Group: "system", Description: "Безопасно сгенерировать SHA-512 password hash", Run: generatePasswordHash},
		{Names: []string{"git-ignore", "-gi"}, Group: "system", Description: "Управление Git core.fileMode", Run: gitIgnorePermissions},
	}
	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func chmodAll(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("команда chmod не принимает аргументы: она работает с текущей директорией")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("не удалось определить текущую директорию: %w", err)
	}
	output.Warn("Рекурсивное изменение прав на 775: %s", cwd)
	if os.Geteuid() == 0 {
		if err := execx.RunPassthrough("chmod", "-R", "775", "."); err != nil {
			return fmt.Errorf("не удалось изменить права: %w", err)
		}
	} else if err := execx.RunPassthrough("sudo", "chmod", "-R", "775", "."); err != nil {
		return fmt.Errorf("не удалось изменить права: %w", err)
	}
	output.Success("Права 775 установлены для текущей директории")
	return nil
}

func generatePasswordHash(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("пароль нельзя передавать аргументом: запустите команду без аргументов")
	}
	if !execx.Exists("openssl") || !execx.Exists("stty") {
		return fmt.Errorf("для генерации хэша нужны openssl и stty")
	}
	fmt.Fprint(os.Stderr, "Пароль: ")
	if err := execx.RunPassthrough("stty", "-echo"); err != nil {
		return fmt.Errorf("не удалось отключить отображение ввода: %w", err)
	}
	defer func() { _ = execx.RunPassthrough("stty", "echo") }()
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return fmt.Errorf("не удалось прочитать пароль: %w", err)
	}
	password = strings.TrimRight(password, "\r\n")
	if password == "" {
		return fmt.Errorf("пароль не может быть пустым")
	}
	result, err := execx.OutputWithInput([]byte(password+"\n"), "openssl", "passwd", "-6", "-stdin")
	password = strings.Repeat("\x00", len(password))
	if err != nil {
		return fmt.Errorf("не удалось создать password hash: %w", err)
	}
	output.Plain("%s", strings.TrimSpace(result.Stdout))
	return nil
}

func gitIgnorePermissions(args []string) error {
	result, err := execx.Output("git", "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(result.Stdout) != "true" {
		return fmt.Errorf("текущая директория не находится внутри Git-репозитория")
	}
	arg := "--disable"
	if len(args) > 0 {
		arg = args[0]
	}
	switch arg {
	case "--status":
		result, err := execx.Output("git", "config", "--get", "core.fileMode")
		value := strings.TrimSpace(result.Stdout)
		if err != nil && value == "" {
			value = "true"
		}
		if value == "false" {
			output.Success("Git не отслеживает изменения прав доступа: core.fileMode=false")
		} else {
			output.Warn("Git отслеживает изменения прав доступа: core.fileMode=%s", fallback(value, "true"))
		}
	case "--enable":
		if err := execx.RunPassthrough("git", "config", "core.fileMode", "true"); err != nil {
			return err
		}
		output.Info("Git теперь отслеживает chmod-изменения")
	case "--disable":
		if err := execx.RunPassthrough("git", "config", "core.fileMode", "false"); err != nil {
			return err
		}
		output.Success("Git больше не отслеживает chmod-изменения")
	case "-h", "--help", "help":
		output.Plain("Использование: qq git-ignore [--disable|--enable|--status]")
	default:
		return fmt.Errorf("неизвестный аргумент: %s", arg)
	}
	return nil
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
