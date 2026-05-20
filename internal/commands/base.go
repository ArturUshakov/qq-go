package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ArturUshakov/qq-go/internal/command"
	"github.com/ArturUshakov/qq-go/internal/output"
	"github.com/ArturUshakov/qq-go/internal/version"
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
			Description: "Генерирует shell completion: bash, zsh, fish",
			Usage:       "qq completion <bash|zsh|fish>",
			Run: func(args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("укажите shell: bash, zsh или fish")
				}
				return printCompletion(registry, args[0])
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
		updateCommand(),
	}
	for _, cmd := range commands {
		if err := registry.Register(cmd); err != nil {
			return err
		}
	}
	return nil
}

func printCompletion(registry *command.Registry, shell string) error {
	names := registry.Names()
	switch shell {
	case "bash":
		fmt.Printf("_qq_completion() {\n")
		fmt.Printf("  local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
		fmt.Printf("  COMPREPLY=( $(compgen -W '%s' -- \"$cur\") )\n", strings.Join(names, " "))
		fmt.Printf("}\ncomplete -F _qq_completion qq\n")
	case "zsh":
		fmt.Printf("#compdef qq\n")
		fmt.Printf("_qq() {\n")
		fmt.Printf("  local -a commands\n")
		fmt.Printf("  commands=(%s)\n", strings.Join(names, " "))
		fmt.Printf("  _describe 'qq commands' commands\n")
		fmt.Printf("}\n_qq \"$@\"\n")
	case "fish":
		for _, name := range names {
			fmt.Printf("complete -c qq -f -a %s\n", name)
		}
	default:
		return fmt.Errorf("неподдерживаемый shell: %s", shell)
	}
	return nil
}

func runDoctor() error {
	output.Title("Проверка окружения")
	output.Plain("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	checkTool("docker")
	checkTool("git")
	checkTool("curl")
	checkTool("php")
	checkTool("openssl")
	checkTool("htpasswd")
	return nil
}
