VERSION := $(shell git describe --tags --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
GOFILES := $(shell find . -not -path './vendor*' -type f -name '*.go')
GOOS := GOOS=linux
GOARCH := GOARCH=amd64
LDFLAGS := -ldflags "-X=go.smartmachine.io/awsci-api/main.Version=$(VERSION) -X=go.smartmachine.io/awsci-api/main.Commit=$(COMMIT)"
TEST_STAMP := .test.stamp

SOURCES := $(wildcard fn/*)
BINARIES := $(subst fn/,,$(SOURCES))
ZIPS := $(addsuffix .zip,$(BINARIES))

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

$(BINARIES): %: fn/%/main.go
	$(info Compiling $@ Lambda)
	@$(GOOS) $(GOARCH) go build -v $(LDFLAGS) ./fn/$@

$(ZIPS): %.zip: %
	$(info Packaging $@)
	@zip $@ $<


.phony: debug
debug: ## Test auto binary functionality
	@echo Sources: $(SOURCES)
	@echo Binaries: $(BINARIES)
	@echo Zips: $(ZIPS)

.phony: build
build: $(BINARIES) ## Build all binary artifacts

.phony: zip
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
