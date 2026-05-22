.PHONY: build test

build:
	go build -o bin/codex-quick-model-switch ./cmd/codex-quick-model-switch

test:
	go test ./...
