# PROJECT_NOTES

Что сделано:

- Python CLI переписан на Go без внешних зависимостей.
- Сохранены короткие алиасы старой версии.
- Добавлены команды `update`, `doctor`, `where`, `completion`.
- Настроены GitHub Actions:
  - `.github/workflows/ci.yml` — gofmt, go vet, go test, build;
  - `.github/workflows/release.yml` — сборка Linux/macOS для amd64/arm64 и публикация GitHub Release.
- Добавлены `scripts/install.sh` и `scripts/uninstall.sh`.
- Добавлен Makefile для локальной сборки.

Важно:

- В коде self-update и install.sh используется репозиторий `ArturUshakov/qq-go`.
- Если итоговый GitHub-репозиторий будет называться иначе, нужно заменить `ArturUshakov/qq-go` в:
  - `internal/commands/update.go`
  - `scripts/install.sh`
  - `go.mod`
  - `README.md`
  - `.github/workflows/release.yml` в ldflags module path

Первый релиз:

```bash
git init
git add .
git commit -m "Rewrite qq in Go"
git branch -M main
git remote add origin git@github.com:ArturUshakov/qq-go.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

После публикации релиза установка:

```bash
curl -fsSL https://raw.githubusercontent.com/ArturUshakov/qq-go/main/scripts/install.sh | sh
```
