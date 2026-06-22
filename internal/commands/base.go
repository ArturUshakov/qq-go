package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/SolasWyrd/qq-go/internal/command"
	"github.com/SolasWyrd/qq-go/internal/execx"
	"github.com/SolasWyrd/qq-go/internal/output"
	"github.com/SolasWyrd/qq-go/internal/version"
)

func RegisterBase(registry *command.Registry) error {
	commands := []command.Command{
		{
			Names:       []string{"help", "-h", "--help"},
			Group:       "base",
			Description: "Выводит справку по командам",
			Run: func(args []string) error {
				command.PrintHelp(registry)
				return nil
			},
		},
		{
			Names:       []string{"version", "--version", "-v"},
			Group:       "base",
			Description: "Показывает версию qq",
			Run: func(args []string) error {
				output.Plain("qq %s", version.Version)
				output.Plain("commit: %s", version.Commit)
				output.Plain("date:   %s", version.Date)
				return nil
			},
		},
		{
			Names:       []string{"completion"},
			Group:       "base",
			Description: "Генерирует или устанавливает shell completion: bash, zsh, fish",
			Usage:       "qq completion [bash|zsh|fish|install]",
			Run: func(args []string) error {
				shell := ""

				if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "install") {
					if len(args) > 1 {
						shell = strings.ToLower(strings.TrimSpace(args[1]))
					} else {
						shell = detectShell()
					}
					return installCompletion(registry, shell)
				}

				if len(args) > 0 {
					shell = strings.ToLower(strings.TrimSpace(args[0]))
				} else {
					shell = detectShell()
				}

				if shell == "" {
					return fmt.Errorf("не удалось определить shell, укажите явно: qq completion bash, qq completion zsh или qq completion fish")
				}

				return printCompletion(registry, shell)
			},
		},
		{
			Names:       []string{"where"},
			Group:       "base",
			Description: "Показывает путь к текущему бинарнику",
			Run: func(args []string) error {
				path, err := os.Executable()
				if err != nil {
					return err
				}
				resolved, err := filepath.EvalSymlinks(path)
				if err != nil {
					resolved = path
				}
				output.Plain(resolved)
				return nil
			},
		},
		{
			Names:       []string{"doctor"},
			Group:       "base",
			Description: "Проверяет окружение и доступность внешних утилит",
			Run: func(args []string) error {
				return runDoctor()
			},
		},
		updateCommand(registry),
	}
	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func detectShell() string {
	shellPath := os.Getenv("SHELL")
	shellName := strings.ToLower(filepath.Base(shellPath))

	switch shellName {
	case "bash", "zsh", "fish":
		return shellName
	default:
		return ""
	}
}

func printCompletion(registry *command.Registry, shell string) error {
	completion, err := completionScript(registry, shell)
	if err != nil {
		return err
	}
	fmt.Print(completion)
	return nil
}

func completionScript(registry *command.Registry, shell string) (string, error) {
	names := registry.AllNames()
	switch shell {
	case "bash":
		return fmt.Sprintf("_qq_completion() {\n  local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n  COMPREPLY=( $(compgen -W '%s' -- \"$cur\") )\n}\ncomplete -F _qq_completion qq\n", strings.Join(names, " ")), nil
	case "zsh":
		return fmt.Sprintf("#compdef qq\n_qq() {\n  local -a commands\n  commands=(%s)\n  _describe 'qq commands' commands\n}\n_qq \"$@\"\n", strings.Join(names, " ")), nil
	case "fish":
		var builder strings.Builder
		for _, name := range names {
			fmt.Fprintf(&builder, "complete -c qq -f -a %s\n", name)
		}
		return builder.String(), nil
	default:
		return "", fmt.Errorf("неподдерживаемый shell: %s", shell)
	}
}

func installCompletion(registry *command.Registry, shell string) error {
	if shell == "" {
		return fmt.Errorf("не удалось определить shell, укажите явно: qq completion install bash, qq completion install zsh или qq completion install fish")
	}

	completion, err := completionScript(registry, shell)
	if err != nil {
		return err
	}

	path, err := completionPath(shell)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(completion), 0644); err != nil {
		return err
	}

	switch shell {
	case "bash":
		if err := ensureBashCompletionLoaded(path); err != nil {
			return err
		}
	case "zsh":
		if err := ensureZshCompletionLoaded(path); err != nil {
			return err
		}
	}

	output.Success("Completion установлен: %s", path)
	output.Info("Откройте новый терминал или перезапустите shell.")
	return nil
}

func completionPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "qq"), nil
	case "zsh":
		return filepath.Join(home, ".zsh", "completions", "_qq"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "qq.fish"), nil
	default:
		return "", fmt.Errorf("неподдерживаемый shell: %s", shell)
	}
}

func ensureBashCompletionLoaded(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".bashrc")
	if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
		configPath = filepath.Join(home, ".bash_profile")
	}

	block := fmt.Sprintf("\n# qq completion\nif [ -f %q ]; then\n  . %q\nfi\n", path, path)
	return appendBlockIfMissing(configPath, "# qq completion", block)
}

func ensureZshCompletionLoaded(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	completionDir := filepath.Dir(path)
	block := fmt.Sprintf("\n# qq completion\nfpath=(%q $fpath)\nautoload -Uz compinit\ncompinit\n", completionDir)
	return appendBlockIfMissing(filepath.Join(home, ".zshrc"), "# qq completion", block)
}

func appendBlockIfMissing(path string, marker string, block string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(content), marker) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(block)
	return err
}

func runDoctor() error {
	output.Title("Проверка окружения")
	output.Plain("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	required := []string{"docker", "git", "openssl", "stty"}
	missing := make([]string, 0)
	for _, tool := range required {
		if execx.Exists(tool) {
			output.Success("✔ %s найден", tool)
		} else {
			output.Warn("⚠ %s не найден", tool)
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("отсутствуют обязательные утилиты: %s", strings.Join(missing, ", "))
	}
	return nil
}
