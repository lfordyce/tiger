GOLANGCI_LINT_VERSION = $(shell head -n 1 .golangci.yml | tr -d '\# ')
TMPDIR ?= /tmp

.PHONY: tests
tests :
	go test -race -timeout 210s ./...

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
