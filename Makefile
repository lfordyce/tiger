GOLANGCI_LINT_VERSION = $(shell head -n 1 .golangci.yml | tr -d '\# ')
TMPDIR ?= /tmp
TEST_FLAGS ?=

.PHONY: run-linter
run-linter:
	@docker run --rm -t -v $(shell pwd):/app \
			-v $(TMPDIR)/golangci-cache-$(GOLANGCI_LINT_VERSION):/golangci-cache \
			--env "GOLANGCI_LINT_CACHE=/golangci-cache" \
			-w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
			make lint

.PHONY: lint
lint :
	golangci-lint run --out-format=tab --new-from-rev main ./...

.PHONY: format
format :
	find . -name '*.go' -exec gofmt -s -w {} +

.PHONY: container
container:
	docker build --rm --pull --no-cache -t lfordyce/tiger .

.PHONY: tests
tests :
	make test-with-flags TEST_FLAGS='-v -race -covermode atomic -bench=. -benchmem -timeout 210s'

.PHONY: test-short
test-short :
	make test-with-flags TEST_FLAGS='-short -race -timeout 210s'

.PHONY: test-with-flags
test-with-flags :
	@go clean -testcache && go test $(TEST_FLAGS) ./...
