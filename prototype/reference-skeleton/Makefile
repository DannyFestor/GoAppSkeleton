.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-z/-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-12s %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the web server
	go run ./cmd/web

.PHONY: build
build: ## Build the binary into bin/
	go build -o bin/web ./cmd/web

.PHONY: test
test: ## Run the tests with the race detector
	go test -race ./...

.PHONY: audit
audit: ## Check formatting, vet, and module tidiness
	test -z "$$(gofmt -l .)"
	go vet ./...
	go mod tidy -diff
