GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOBIN=$(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
OSARCH=$(shell uname -m)
GOLANGCI_LINT_PATH=$(GOBIN)/golangci-lint

.PHONY: install-git-hooks
install-git-hooks:
	@echo "Installing git hooks..."
	pre-commit install --hook-type pre-commit
	pre-commit install --hook-type commit-msg

.PHONY: install-golangci-lint
install-golangci-lint:
	@echo "Installing github.com/golangci/golangci-lint..."
	@(test -f $(GOLANGCI_LINT_PATH) && echo "github.com/golangci/golangci-lint is already installed. Skipping...") || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.57.2

.PHONY: install-tools
install-tools: install-golangci-lint

.PHONY: install
install: install-tools install-git-hooks
