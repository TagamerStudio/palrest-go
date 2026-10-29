.PHONY: test lint vet fmt fuzz tidy check help

GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT_CMD = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOFUMPT_VERSION ?= v0.12.0
GOIMPORTS_VERSION ?= v0.52.0

test:
	go test -count=1 -race -v ./...

lint:
	$(GOLANGCI_LINT_CMD) run --config .golangci.yml --timeout 5m ./...

vet:
	go vet ./...

fuzz:
	@targets="$$(go test -list '^Fuzz' . | grep '^Fuzz')"; \
	if [ -z "$$targets" ]; then echo "no fuzz targets found" >&2; exit 1; fi; \
	for target in $$targets; do \
		go test -run=^$$ -fuzz="^$$target$$" -fuzztime=30s . || exit 1; \
	done

fmt:
	gofmt -s -w .
	go run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION) -w .
	go run mvdan.cc/gofumpt@$(GOFUMPT_VERSION) -w .

tidy:
	go mod tidy

check: vet lint test

help:
	@echo 'Targets: test lint vet fmt fuzz tidy check help'
