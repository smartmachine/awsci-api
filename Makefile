VERSION := $(shell git describe --tags --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
GOFILES := $(shell find . -not -path './vendor*' -type f -name '*.go')
GOOS := GOOS=linux
GOARCH := GOARCH=amd64
LDFLAGS := -ldflags "-X=go.smartmachine.io/awsci-api/main.Version=$(VERSION) -X=go.smartmachine.io/awsci-api/main.Commit=$(COMMIT)"
TEST_STAMP := .test.stamp

.phony: all
all: dep build zip ## Generate and build everything

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

login: $(GOFILES)
	$(info Compiling Login Lambda)
	@$(GOOS) $(GOARCH) go build -v $(LDFLAGS) ./fn/login

.phony: build
build: login ## Build all binary artifacts

login.zip: login
	@zip login.zip login

.phony: zip
zip: login.zip ## Package the Lambda for distribution

.phony: clean
clean: ## Clean all build artifacts
	$(info Cleaning all build artifacts)
	@rm -rf login .test.stamp login.zip
	@go clean

.phony: veryclean
veryclean: clean ## Clean all caches and generated objects
	@go clean -cache -testcache -modcache

.phony: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
