VERSION := $(shell git describe --tags --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
GOFILES := $(shell find . -not -path './vendor*' -type f -name '*.go')
GOOS := GOOS=linux
GOARCH := GOARCH=amd64
LDFLAGS := -ldflags "-X=go.smartmachine.io/awsci-api/main.Version=$(VERSION) -X=go.smartmachine.io/awsci-api/main.Commit=$(COMMIT)"
TEST_STAMP := .test.stamp

.phony: all
all: dep test build zip ## Generate and build everything

.phony: test
test: $(TEST_STAMP) ## Run unit tests

.phony: dep
dep: ## Make sure all dependencies are up to date
	@go mod tidy
	@go mod vendor

$(TEST_STAMP): $(GOFILES)
	$(info Running unit tests)
	@go test ./...
	@touch $@

awsci-api: $(GOFILES)
	$(info Compiling project)
	@$(GOOS) $(GOARCH) go build -v $(LDFLAGS)

.phony: build
build: awsci-api ## Build all binary artifacts

awsci-api.zip: awsci-api
	@zip awsci-api.zip awsci-api

.phony: zip
zip: awsci-api.zip ## Package the Lambda for distribution

.phony: clean
clean: ## Clean all build artifacts
	$(info Cleaning all build artifacts)
	@rm -rf awsci-api .test.stamp awsci-api.zip
	@go clean

.phony: veryclean
veryclean: clean ## Clean all caches and generated objects
	@go clean -cache -testcache -modcache

.phony: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
