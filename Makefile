VERSION := $(shell git describe --tags --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
GOFILES := $(shell find . -not -path './vendor*' -type f -name '*.go')
GOOS := GOOS=linux
GOARCH := GOARCH=amd64
TEST_STAMP := .test.stamp

SOURCES = $(wildcard fn/*/*/main.go)
BINPATHS = $(subst /main.go,,$(subst fn/,,$(SOURCES)))
BINARIES = $(subst /,-,$(BINPATHS))
ZIPS = $(addsuffix .zip,$(BINARIES))

.phony: all
all: dep build zip ## Generate and build everything

.phony: test
test: $(TEST_STAMP) ## Run unit tests

.phony: dep
dep: ## Make sure all dependencies are up to date
	@go mod tidy

$(TEST_STAMP): $(GOFILES)
	$(info Running unit tests)
	@go test ./...
	@touch $@

.SECONDEXPANSION:
$(BINARIES): fn/$$(subst -,/,$$@)/main.go
	$(info Compiling $@ lambda)
	@$(GOOS) $(GOARCH) go build -o $@ ./$(subst /main.go,,$<)

$(ZIPS): %.zip: %
	$(info Packaging $@)
	@zip $@ $<

build: $(BINARIES) ## Build all binary artifacts

zip: $(ZIPS) ## Package the Lambda functions for distribution

.phony: clean
clean: ## Clean all build artifacts
	$(info Cleaning all build artifacts)
	@rm -rf $(BINARIES) $(ZIPS) .test.stamp
	@go clean

.phony: veryclean
veryclean: clean ## Clean all caches and generated objects
	@go clean -cache -testcache -modcache

.phony: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
