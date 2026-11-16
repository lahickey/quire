# Makefile for ledgerd: build, test, package.
# `make help` lists the targets.

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables --no-builtin-rules

BINARY   := ledgerd
PKG      := example.com/acme/ledgerd
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD)
BUILD_AT  = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

GO       ?= go
GOFLAGS  += -trimpath
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
PLATFORMS := linux/amd64 linux/arm64 darwin/arm64

SRC      := $(shell find . -name '*.go' -not -path './vendor/*')
TEMPLATES := $(wildcard web/templates/*.html)
DIST     := dist
OBJ      := $(patsubst cmd/%/main.go,$(DIST)/%,$(wildcard cmd/*/main.go))

ifeq ($(OS),Windows_NT)
  EXE := .exe
else
  EXE :=
  UNAME := $(shell uname -s)
endif

ifdef CI
  TEST_FLAGS := -race -count=1 -coverprofile=cover.out
else
  TEST_FLAGS := -count=1
endif

define banner
	@printf '\033[1m==> %s\033[0m\n' "$(1)"
endef

.PHONY: all build test lint fmt clean dist docker help install

all: lint test build ## Lint, test and build

build: $(DIST)/$(BINARY)$(EXE) ## Build the binary

$(DIST)/$(BINARY)$(EXE): $(SRC) go.mod go.sum | $(DIST)
	$(call banner,building $@)
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS) -X main.built=$(BUILD_AT)' -o $@ ./cmd/$(BINARY)

$(DIST):
	@mkdir -p $@

# One archive per platform: dist/ledgerd-linux-amd64.tar.gz
dist: $(foreach p,$(PLATFORMS),$(DIST)/$(BINARY)-$(subst /,-,$(p)).tar.gz) ## Cross-compile and archive

$(DIST)/$(BINARY)-%.tar.gz: $(SRC) | $(DIST)
	$(eval os := $(word 1,$(subst -, ,$*)))
	$(eval arch := $(word 2,$(subst -, ,$*)))
	GOOS=$(os) GOARCH=$(arch) CGO_ENABLED=0 \
		$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/$*/$(BINARY) ./cmd/$(BINARY)
	tar -C $(DIST)/$* -czf $@ $(BINARY)
	@rm -rf $(DIST)/$*

test: ## Run the tests
	$(GO) test $(TEST_FLAGS) ./...
ifdef CI
	@$(GO) tool cover -func=cover.out | tail -n 1
endif

lint: ## Vet and lint
	$(GO) vet ./...
	@command -v golangci-lint >/dev/null || { echo "golangci-lint is not installed" >&2; exit 1; }
	golangci-lint run --timeout 3m

fmt: ## Format the source
	gofmt -w -s $(SRC)
	@for f in $(TEMPLATES); do \
		echo "checking $$f"; \
		test -s "$$f" || exit 1; \
	done

docker: ## Build the image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(BINARY):$(VERSION) -t $(BINARY):latest .

install: build ## Install into $(PREFIX)/bin
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(DIST)/$(BINARY)$(EXE) $(DESTDIR)$(PREFIX)/bin/

clean: ## Remove what was built
	-rm -rf $(DIST) cover.out

help: ## Show this help
	@awk 'BEGIN { FS = ":.*## " } /^[a-zA-Z_-]+:.*## / { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

PREFIX ?= /usr/local
export GOFLAGS
-include local.mk
