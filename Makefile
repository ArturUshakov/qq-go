APP_NAME=qq
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-s -w -X github.com/SolasWyrd/qq-go/internal/version.Version=$(VERSION) -X github.com/SolasWyrd/qq-go/internal/version.Commit=$(COMMIT) -X github.com/SolasWyrd/qq-go/internal/version.Date=$(DATE)

.PHONY: build test lint clean release-local check-release-version check-git-clean release

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/qq

test:
	go test ./...

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -rf bin dist

release-local: clean
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/qq ./cmd/qq
	tar -C dist -czf dist/qq_linux_amd64.tar.gz qq
	rm dist/qq
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/qq ./cmd/qq
	tar -C dist -czf dist/qq_linux_arm64.tar.gz qq
	rm dist/qq
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/qq ./cmd/qq
	tar -C dist -czf dist/qq_darwin_amd64.tar.gz qq
	rm dist/qq
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/qq ./cmd/qq
	tar -C dist -czf dist/qq_darwin_arm64.tar.gz qq
	rm dist/qq
	cd dist && shasum -a 256 *.tar.gz > checksums.txt

check-release-version:
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Укажите версию: make release VERSION=v0.1.0"; \
		exit 1; \
	fi
	@case "$(VERSION)" in \
		v[0-9]*.[0-9]*.[0-9]*) ;; \
		*) echo "VERSION должен быть в формате vX.Y.Z, например v0.1.0"; exit 1 ;; \
	esac
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "Тег $(VERSION) уже существует"; \
		exit 1; \
	fi

check-git-clean:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Рабочее дерево не чистое. Сначала закоммитьте или уберите изменения."; \
		git status --short; \
		exit 1; \
	fi

release: check-release-version check-git-clean test
	git tag -a "$(VERSION)" -m "Release $(VERSION)"
	git push origin "$(VERSION)"
	@echo "Тег $(VERSION) опубликован. GitHub Actions соберёт и опубликует release."
