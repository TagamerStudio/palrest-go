.PHONY: test lint fmt tidy check help

GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT_CMD = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOFUMPT_VERSION ?= v0.7.0
GOIMPORTS_VERSION ?= v0.30.0

test:
	go test -count=1 -race -v ./...

lint:
	$(GOLANGCI_LINT_CMD) run --config .golangci.yml --timeout 5m ./...

fmt:
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	gofmt -s -w .
	$(shell go env GOPATH)/bin/goimports -w .
	$(shell go env GOPATH)/bin/gofumpt -w .

tidy:
	go mod tidy

check: lint test

help:
	@echo 'Targets: test lint fmt tidy check help'
