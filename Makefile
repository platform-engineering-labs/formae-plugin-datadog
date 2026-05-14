# Formae Datadog Plugin Makefile

# Plugin metadata - extracted from formae-plugin.pkl
PLUGIN_NAME := $(shell pkl eval -x 'name' formae-plugin.pkl 2>/dev/null || echo "datadog")
PLUGIN_VERSION := $(shell pkl eval -x 'version' formae-plugin.pkl 2>/dev/null || echo "0.1.0")
PLUGIN_NAMESPACE := $(shell pkl eval -x 'namespace' formae-plugin.pkl 2>/dev/null || echo "DATADOG")

# Build settings
GO := go
GOFLAGS := -trimpath
BINARY := $(PLUGIN_NAME)

# Installation paths
# Plugin discovery expects lowercase directory names matching the plugin name
PLUGIN_BASE_DIR := $(HOME)/.pel/formae/plugins
INSTALL_DIR := $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)/v$(PLUGIN_VERSION)

.PHONY: all build test test-unit test-integration lint verify-schema gen-pkl clean install help clean-environment conformance-test conformance-test-crud conformance-test-discovery conformance-test-crud-run conformance-test-discovery-run

all: build

## build: Build the plugin binary
build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY) .

## test: Run all tests (excludes integration/conformance)
test:
	$(GO) test -v ./...

## test-unit: Run unit tests only
test-unit:
	$(GO) test -v -tags=unit ./...

## test-integration: Run integration tests (requires Datadog credentials)
## Env: DD_API_KEY, DD_APP_KEY, DD_SITE
test-integration:
	$(GO) test -v -tags=integration -timeout 15m ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## verify-schema: Validate PKL schema files against formae conventions
verify-schema:
	$(GO) run github.com/platform-engineering-labs/formae/pkg/plugin/testutil/cmd/verify-schema --namespace $(PLUGIN_NAMESPACE) ./schema/pkl

## gen-pkl: Resolve all PKL project dependencies
gen-pkl:
	pkl project resolve schema/pkl
	pkl project resolve examples/basic
	pkl project resolve testdata

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/

## install: Build and install plugin locally (binary + schema + manifest)
## Installs to ~/.pel/formae/plugins/<name>/v<version>/
## Removes any existing versions of the plugin first to ensure clean state.
install: build
	@echo "Installing $(PLUGIN_NAME) v$(PLUGIN_VERSION) (namespace: $(PLUGIN_NAMESPACE))..."
	@rm -rf $(PLUGIN_BASE_DIR)/$(PLUGIN_NAME)
	@mkdir -p $(INSTALL_DIR)/schema/pkl
	@cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@cp -r schema/pkl/* $(INSTALL_DIR)/schema/pkl/
	@if [ -f schema/Config.pkl ]; then cp schema/Config.pkl $(INSTALL_DIR)/schema/; fi
	@cp formae-plugin.pkl $(INSTALL_DIR)/
	@echo "Installed to $(INSTALL_DIR)"
	@echo "  - Binary: $(INSTALL_DIR)/$(BINARY)"
	@echo "  - Schema: $(INSTALL_DIR)/schema/pkl/"
	@echo "  - Manifest: $(INSTALL_DIR)/formae-plugin.pkl"

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## clean-environment: Clean up test resources in Datadog account
## Deletes monitors matching the test prefix via the Datadog API.
clean-environment:
	@./scripts/ci/clean-environment.sh

## conformance-test: Run all conformance tests (CRUD + discovery)
## Usage: make conformance-test [TEST=monitor] [PARALLEL=1] [TIMEOUT=15]
## Calls clean-environment before and after tests.
##
## Parameters:
##   TEST     - Filter tests by name pattern (e.g., TEST=monitor)
##   PARALLEL - Concurrent tests inside the SDK (default: 1 = sequential)
##   TIMEOUT  - Test timeout in minutes (default: 15)
##
## The conformance SDK installs the latest released formae via orbital
## unless FORMAE_BINARY is set (e.g. by nightly which builds from source).
conformance-test: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@$(MAKE) conformance-test-crud-run conformance-test-discovery-run TEST=$(TEST) PARALLEL=$(PARALLEL) TIMEOUT=$(TIMEOUT); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-crud: Run CRUD tests with cleanup (convenience for local dev)
conformance-test-crud: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@$(MAKE) conformance-test-crud-run TEST=$(TEST) PARALLEL=$(PARALLEL) TIMEOUT=$(TIMEOUT); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-discovery: Run discovery tests with cleanup (convenience for local dev)
conformance-test-discovery: install
	@echo "Pre-test cleanup..."
	@./scripts/ci/clean-environment.sh || true
	@echo ""
	@$(MAKE) conformance-test-discovery-run TEST=$(TEST) PARALLEL=$(PARALLEL) TIMEOUT=$(TIMEOUT); \
	TEST_EXIT=$$?; \
	echo ""; \
	echo "Post-test cleanup..."; \
	./scripts/ci/clean-environment.sh || true; \
	exit $$TEST_EXIT

## conformance-test-crud-run: Run only CRUD lifecycle tests (no cleanup)
## Used by CI matrix jobs where cleanup is managed separately.
conformance-test-crud-run:
	@echo "Running CRUD conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=crud FORMAE_TEST_PARALLEL="$(PARALLEL)" \
		$(GO) test -tags=conformance -v -timeout $(or $(TIMEOUT),15)m ./...

## conformance-test-discovery-run: Run only discovery tests (no cleanup)
## Used by CI matrix jobs where cleanup is managed separately.
conformance-test-discovery-run:
	@echo "Running discovery conformance tests..."
	@FORMAE_TEST_FILTER="$(TEST)" FORMAE_TEST_TYPE=discovery FORMAE_TEST_PARALLEL="$(PARALLEL)" \
		$(GO) test -tags=conformance -v -timeout $(or $(TIMEOUT),15)m ./...
