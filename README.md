# qq

`qq` — небольшая CLI-утилита для ежедневной работы с Docker, Git и локальным окружением.

Проект переписан на Go и поставляется как один бинарник для Linux и macOS.

## Возможности

- один бинарный файл без Python и внешнего runtime;
- Linux/macOS, amd64/arm64;
- команды для Docker-контейнеров;
- очистка Docker-ресурсов;
- генерация password hash через доступные системные утилиты;
- управление `git config core.fileMode`;
- self-update через GitHub Releases;
- shell completion для Bash, Zsh и Fish;
- GitHub Actions для CI и релизов.

## Установка

```bash
curl -fsSL https://raw.githubusercontent.com/ArturUshakov/qq-go/main/scripts/install.sh | sh
```

По умолчанию бинарник устанавливается в:

```text
~/.local/bin/qq
```

Можно изменить директорию установки:

```bash
QQ_INSTALL_DIR="$HOME/.qq/bin" sh scripts/install.sh
```

## Обновление

```bash
qq update
```

Команда скачивает последний GitHub Release, выбирает архив под текущую ОС/архитектуру и заменяет текущий бинарник.

## Основные команды

```bash
qq help
qq version
qq doctor
qq where
qq update
```

## Docker-команды

Показать активные проекты:

```bash
qq list
qq -l
```

Остановить все контейнеры:

```bash
qq down
qq -d
```

Остановить контейнеры по части имени или compose project:

```bash
qq down api
```

Войти в контейнер по части имени:

```bash
qq exec app
qq -e app
```

Войти root-пользователем:

```bash
qq exec app -r
```

Выполнить конкретную команду:

```bash
qq exec app php -v
```

Проверить локальный IP и доступные HTTP-порты контейнеров:

```bash
qq network
qq -net
```

## Очистка Docker

Dry-run без удаления:

```bash
qq clear --dry-run
```

Очистка без подтверждения:

```bash
qq clear --yes
```

Подробный вывод Docker-команд:

```bash
qq clear --verbose
```

Только dangling images:

```bash
qq cleanup-docker-images
qq -dni
```

Только builder cache:

```bash
qq prune-builder
qq -pb
```

## Системные команды

Сгенерировать password hash:

```bash
qq generate-password-hash 'secret'
qq -gph 'secret'
```

Показать локальный IP:

```bash
qq external-ip
qq -eip
```

Отключить отслеживание chmod-изменений в текущем Git-репозитории:

```bash
qq git-ignore
qq -gi
```

Проверить состояние:

```bash
qq git-ignore --status
```

## Completion

Bash:

```bash
mkdir -p ~/.local/share/bash-completion/completions
qq completion bash > ~/.local/share/bash-completion/completions/qq
```

Zsh:

```bash
mkdir -p ~/.zsh/completions
qq completion zsh > ~/.zsh/completions/_qq
printf '\nfpath=(~/.zsh/completions $fpath)\nautoload -Uz compinit\ncompinit\n' >> ~/.zshrc
```

Fish:

```bash
mkdir -p ~/.config/fish/completions
qq completion fish > ~/.config/fish/completions/qq.fish
```

## Разработка

```bash
go test ./...
go build -o bin/qq ./cmd/qq
make build
```

Локальная сборка релизных архивов:

```bash
make release-local VERSION=v0.1.0
```

## Релиз

Создать тег:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions соберет архивы:

```text
qq_linux_amd64.tar.gz
qq_linux_arm64.tar.gz
qq_darwin_amd64.tar.gz
qq_darwin_arm64.tar.gz
checksums.txt
install.sh
uninstall.sh
```

## Совместимость с предыдущими алиасами

Сохранены короткие команды из Python-версии:

```text
-l    list
-d    down
-e    exec
-net  network
-dni  cleanup-docker-images
-pb   prune-builder
-clr  clear
-ch   chmod
-gph  generate-password-hash
-eip  external-ip
-gi   git-ignore
-h    help
```
