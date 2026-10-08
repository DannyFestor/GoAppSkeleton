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
test: test-unit test-feature test-e2e ## Run the unit, feature, and e2e tiers

.PHONY: test-unit
test-unit: ## Run the unit tests
	go test -race ./...

.PHONY: test-feature
test-feature: ## Run the feature tests
	go test -race -tags feature ./tests/feature/...

.PHONY: test-e2e
test-e2e: ## Run the e2e tests against the built binary
	go test -race -tags e2e ./tests/e2e/...

.PHONY: audit
audit: ## Check formatting, vet, and module tidiness
	test -z "$$(gofmt -l .)"
	go vet ./...
	go mod tidy -diff
