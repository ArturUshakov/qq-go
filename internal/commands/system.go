package commands

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/execx"
	"github.com/ArturUshakov/qq-go/internal/output"
)

func RegisterSystem(registry *command.Registry) error {
	commands := []command.Command{
		{Names: []string{"chmod", "-ch"}, Group: "system", Description: "Рекурсивно выставить chmod 777 в текущей директории", Run: chmodAll},
		{Names: []string{"generate-password-hash", "-gph"}, Group: "system", Description: "Сгенерировать парольный хэш через htpasswd/php/openssl", Run: generatePasswordHash},
		{Names: []string{"external-ip", "-eip"}, Group: "system", Description: "Показать локальный IP для внешнего доступа", Run: externalIP},
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
	output.Warn("Изменение прав доступа на 777 для текущей директории")
	if err := execx.RunPassthrough("sudo", "chmod", "777", "-R", "."); err != nil {
		return fmt.Errorf("ошибка при изменении прав доступа: %w", err)
	}
	output.Success("Права успешно обновлены")
	return nil
}

func generatePasswordHash(args []string) error {
	if len(args) == 0 || args[0] == "" {
		return fmt.Errorf("укажите строку для генерации хэша")
	}
	password := args[0]
	tools := []struct {
		name string
		args []string
	}{
		{"htpasswd", []string{"-bnBC", "10", "", password}},
		{"php", []string{"-r", "echo password_hash(" + shellPHPString(password) + ", PASSWORD_DEFAULT);"}},
		{"openssl", []string{"passwd", "-6", password}},
	}
	for _, tool := range tools {
		if !execx.Exists(tool.name) {
			continue
		}
		result, err := execx.Output(tool.name, tool.args...)
		if err != nil {
			continue
		}
		hash := strings.TrimSpace(result.Stdout)
		if tool.name == "htpasswd" {
			parts := strings.SplitN(hash, ":", 2)
			if len(parts) == 2 {
				hash = parts[1]
			}
		}
		if hash != "" {
			output.Success("Сгенерированный хэш: %s", hash)
			return nil
		}
	}
	return fmt.Errorf("команды htpasswd, php и openssl не найдены или не смогли сгенерировать хэш")
}

func externalIP(args []string) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, item := range interfaces {
		if item.Flags&net.FlagUp == 0 || item.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := item.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			output.Success("IP для внешнего доступа: %s", ip.String())
			return nil
		}
	}
	return fmt.Errorf("не удалось определить IP-адрес")
}

func gitIgnorePermissions(args []string) error {
	if _, err := os.Stat(".git"); err != nil {
		return fmt.Errorf("это не Git-репозиторий: папка .git не найдена")
	}
	arg := "--disable"
	if len(args) > 0 {
		arg = args[0]
	}
	switch arg {
	case "--status":
		result, err := execx.Output("git", "config", "--get", "core.fileMode")
		if err != nil && strings.TrimSpace(result.Stdout) == "" {
			output.Warn("Git отслеживает изменения прав доступа: core.fileMode=true")
			return nil
		}
		value := strings.TrimSpace(result.Stdout)
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

func shellPHPString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}

func fallback(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
