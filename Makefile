SHELL := /usr/bin/env bash
GO ?= go
ZOLA ?= zola
BINARY ?= bin/clusterlog
CLUSTERLOG_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X example.org/bitacora-cluster/internal/clusterlog.Version=$(CLUSTERLOG_VERSION)

.PHONY: all build test vet e2e theme sync sync-check validate validate-local site serve smoke ci clean runme-install runme-list runme-check runme-ci

all: build

build:
	mkdir -p $(dir $(BINARY))
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/clusterlog

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

e2e: build
	./scripts/e2e-local.sh

theme:
	./scripts/bootstrap-theme.sh

sync: build
	./$(BINARY) --root . sync

sync-check: build
	./$(BINARY) --root . sync --check

validate: build
	./$(BINARY) --root . validate --require-zola

validate-local: build
	./$(BINARY) --root . validate --skip-zola

site: theme sync validate
	$(ZOLA) build

serve: theme sync
	$(ZOLA) serve --interface 127.0.0.1

smoke: build
	./scripts/smoke.sh

ci: theme test vet e2e build sync sync-check validate
	$(ZOLA) build

clean:
	rm -rf bin public themes/zola.386

runme-install:
	./scripts/runme-install.sh

runme-list:
	runme ls --filename RUNBOOK.md

runme-check: build
	./scripts/runme-check.sh

runme-ci: build
	runme run --filename RUNBOOK.md --tag ci-safe --all --skip-prompts
