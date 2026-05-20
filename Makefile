APP_NAME=qq
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-s -w -X github.com/ArturUshakov/qq-go/internal/version.Version=$(VERSION) -X github.com/ArturUshakov/qq-go/internal/version.Commit=$(COMMIT) -X github.com/ArturUshakov/qq-go/internal/version.Date=$(DATE)

.PHONY: build test lint clean release-local

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
