.DEFAULT_GOAL := help

.PHONY: test
test: ## Run all unit tests with race detector.
	go test -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires live exchange connectivity).
	go test -race -tags=integration ./integration_tests/

.PHONY: count-go
count-go: ## Count number of lines of all go codes.
	find . -path ./_cmd -prune -o -name "*.go" -type f -print | xargs wc -l | tail -n 1

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'