.PHONY: test lint vet fmt fuzz tidy check help

GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT_CMD = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOFUMPT_VERSION ?= v0.7.0
GOIMPORTS_VERSION ?= v0.30.0
FUZZ_TARGETS = FuzzNormalizeBaseURL FuzzActorUnmarshalJSON FuzzDecodeResponseBody

test:
	go test -count=1 -race -v ./...

lint:
	$(GOLANGCI_LINT_CMD) run --config .golangci.yml --timeout 5m ./...

vet:
	go vet ./...

fuzz:
	for target in $(FUZZ_TARGETS); do \
		go test -run=^$$ -fuzz=$$target -fuzztime=30s . || exit 1; \
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
