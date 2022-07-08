GOLANGCI_LINT_VERSION = $(shell head -n 1 .golangci.yml | tr -d '\# ')
TMPDIR ?= /tmp
TEST_FLAGS ?=

.DEFAULT_GOAL := help

FORMATTING_BEGIN_YELLOW = \033[0;33m
FORMATTING_BEGIN_BLUE = \033[36m
FORMATTING_END = \033[0m

help:
	@printf -- "${FORMATTING_BEGIN_BLUE}                                __________________________  ${FORMATTING_END}\n"
	@printf -- "${FORMATTING_BEGIN_BLUE}                               /_  __/  _/ ____/ ____/ __ \ ${FORMATTING_END}\n"
	@printf -- "${FORMATTING_BEGIN_BLUE}                                / /  / // / __/ __/ / /_/ / ${FORMATTING_END}\n"
	@printf -- "${FORMATTING_BEGIN_BLUE}                               / / _/ // /_/ / /___/ _, _/  ${FORMATTING_END}\n"
	@printf -- "${FORMATTING_BEGIN_BLUE}                              /_/ /___/\____/_____/_/ |_|   ${FORMATTING_END}\n"
	@printf -- "\n"
	@printf -- "                                       T I G E R\n"
	@printf -- "\n"
	@printf -- "---------------------------------------------------------------------------------------\n"
	@printf -- "\n"
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make ${FORMATTING_BEGIN_BLUE}<target>${FORMATTING_END}\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  ${FORMATTING_BEGIN_BLUE}%-46s${FORMATTING_END} %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Building

.PHONY: container
container: ## Build docker image
	docker build --rm --pull --no-cache -t lfordyce/tiger .

.PHONY: timescaledb
timescaledb: ## Run the timescaledb docker-compose file for initial data setup
	@docker compose --file=docker-compose.yml up -d

.PHONY: destroy-timescaledb
destroy-timescaledb: ## Shutdown timescaledb container
	@docker compose --file=docker-compose.yml down -v

##@ Checking

.PHONY: run-linter
run-linter: ## Run golangci linter with docker container against .gloanci.yaml configuration
	@docker run --rm -t -v $(shell pwd):/app \
			-v $(TMPDIR)/golangci-cache-$(GOLANGCI_LINT_VERSION):/golangci-cache \
			--env "GOLANGCI_LINT_CACHE=/golangci-cache" \
			-w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
			make lint

.PHONY: lint
lint: ## Run linter
	golangci-lint run --out-format=tab --new-from-rev main ./...

.PHONY: format
format: ## Format go code
	find . -name '*.go' -exec gofmt -s -w {} +

##@ Testing

.PHONY: tests
tests: ## Runs test including integration and benchmark
	make test-with-flags TEST_FLAGS='-v -race -covermode atomic -bench=. -benchmem -timeout 210s'

.PHONY: test-short
test-short: ## Run the unit test suite
	make test-with-flags TEST_FLAGS='-short -race -timeout 210s'

.PHONY: test-with-flags
test-with-flags: ## Helper target for testing
	@go clean -testcache && go test $(TEST_FLAGS) ./...

