SHELL := /bin/bash
PROJECT=mole
GOPATH ?= $(shell go env GOPATH)

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif

GOENV   	    			:= GO111MODULE=on CGO_ENABLED=0
GO                  := $(GOENV) go
GOBUILD             := $(GO) build $(BUILD_FLAG)
GOTEST              := $(GO) test -v --count=1 --parallel=1 -p=1
GORUN               := $(GO) run
TEST_LDFLAGS        := ""

PACKAGE_LIST        := go list ./...| grep -vE "cmd"
PACKAGES            := $$($(PACKAGE_LIST))

# Targets
.PHONY: cli test

CURDIR := $(shell pwd)
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
export PATH := $(CURDIR)/bin/:$(PATH)

# run starts the server with dev config

cli: lint
	$(GOBUILD) -o bin/mole ./cmd

test:
	$(GOTEST) ./...

## Run golangci-lint linter
lint: golangci-lint
	$(GOLANGCI_LINT) --timeout 5m0s run ./...

## Run golangci-lint linter and perform fixes
lint-fix: golangci-lint
	$(GOLANGCI_LINT) run --fix ./...

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.41.1)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
