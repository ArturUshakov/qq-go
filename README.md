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
curl -fsSL https://raw.githubusercontent.com/SolasWyrd/qq-go/main/scripts/install.sh | sh
```

По умолчанию бинарник устанавливается в:

```text
/usr/local/bin/qq
```

Установка требует права записи в `/usr/local/bin` или доступ к `sudo`. Архив проверяется по SHA-256 перед установкой.

## Обновление

```bash
qq update
```

Команда скачивает последний GitHub Release, проверяет SHA-256 и атомарно устанавливает бинарник в `/usr/local/bin/qq`.

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

Сгенерировать password hash без передачи пароля через аргументы процесса:

```bash
qq generate-password-hash
qq -gph
```


Выставить права `775` рекурсивно для текущей директории:

```bash
qq chmod
```

Команда намеренно применяется ко всему содержимому текущего каталога. Перед запуском проверьте `pwd`.

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

Автоматическая установка для текущего shell:

```bash
qq completion install
```

Можно указать shell явно:

```bash
qq completion install bash
qq completion install zsh
qq completion install fish
```

`scripts/install.sh` запускает установку completion автоматически после установки бинарника.

Ручная генерация:

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

Опубликовать новый тег и запустить GitHub Actions release:

```bash
make release VERSION=v0.1.0
```
