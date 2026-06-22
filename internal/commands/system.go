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
		{Names: []string{"generate-password-hash", "-gph"}, Group: "system", Description: "Безопасно сгенерировать password hash", Run: generatePasswordHash},
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
	if !execx.Exists("stty") {
		return fmt.Errorf("для безопасного ввода пароля требуется stty")
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("не удалось открыть терминал для безопасного ввода пароля: %w", err)
	}
	defer tty.Close()

	fmt.Fprint(tty, "Пароль: ")
	if err := execx.RunWithTerminal(tty, "stty", "-echo"); err != nil {
		return fmt.Errorf("не удалось отключить отображение ввода: %w", err)
	}
	defer func() { _ = execx.RunWithTerminal(tty, "stty", "echo") }()

	password, err := bufio.NewReader(tty).ReadString('\n')
	fmt.Fprintln(tty)
	if err != nil {
		return fmt.Errorf("не удалось прочитать пароль: %w", err)
	}
	password = strings.TrimRight(password, "\r\n")
	if password == "" {
		return fmt.Errorf("пароль не может быть пустым")
	}

	hash, err := createPasswordHash(password)
	password = strings.Repeat("\x00", len(password))
	if err != nil {
		return err
	}

	output.Plain("%s", hash)
	return nil
}

func createPasswordHash(password string) (string, error) {
	input := []byte(password + "\n")

	if execx.Exists("htpasswd") {
		result, err := execx.OutputWithInput(input, "htpasswd", "-niBC", "12", "")
		if err == nil {
			hash := strings.TrimSpace(result.Stdout)
			if index := strings.IndexByte(hash, ':'); index >= 0 {
				hash = strings.TrimSpace(hash[index+1:])
			}
			if hash != "" {
				return hash, nil
			}
		}
	}

	if execx.Exists("php") {
		result, err := execx.OutputWithInput(
			input,
			"php",
			"-r",
			"$password = rtrim(stream_get_contents(STDIN), \"\\r\\n\"); echo password_hash($password, PASSWORD_BCRYPT, ['cost' => 12]), PHP_EOL;",
		)
		if err == nil {
			hash := strings.TrimSpace(result.Stdout)
			if hash != "" {
				return hash, nil
			}
		}
	}

	if execx.Exists("openssl") {
		result, err := execx.OutputWithInput(input, "openssl", "passwd", "-6", "-stdin")
		if err == nil {
			hash := strings.TrimSpace(result.Stdout)
			if hash != "" {
				return hash, nil
			}
		}
	}

	return "", fmt.Errorf("не удалось создать password hash: установите htpasswd или PHP; OpenSSL должен поддерживать 'passwd -6'")
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
