# goalert development targets. Run `make` or `make help` to list them.
.DEFAULT_GOAL := help
.PHONY: help fmt lint vet test build run docker-build helm-lint fmt-check check clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format all Go sources (golangci-lint fmt)
	golangci-lint fmt ./...

lint: ## Lint all Go sources (golangci-lint)
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

test: ## Run all tests with the race detector
	go test -race -count=1 ./...

build: ## Build the bot binary to bin/alert
	go build -o bin/alert ./cmd/goalert-bot

run: ## Run the bot locally (requires CHAT_DOMAIN, AUTH_TOKEN, ...)
	go run ./cmd/goalert-bot

docker-build: ## Build the Docker image (phntom/goalert:dev)
	docker build -t phntom/goalert:dev .

helm-lint: ## Lint the Helm chart strictly
	helm lint --strict charts/goalert

fmt-check: ## Verify Go formatting without changing files
	golangci-lint fmt --diff ./...

check: fmt-check lint vet test build ## CI-equivalent gate: fmt-check, lint, vet, test, build

clean: ## Remove build artifacts (bin/)
	rm -rf bin
