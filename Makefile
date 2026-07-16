.PHONY: help all build test test-race test-coverage test-verbose lint fmt vet clean deps tidy check examples docs install-tools benchmark modernize

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet
BINARY_NAME=langgraphgo
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Colors for terminal output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

# Default target
all: modernize check test build

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)LangGraphGo - Makefile Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET)"
	@echo "  make $(COLOR_GREEN)<target>$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@grep -E '^## ' Makefile | sed 's/## /  $(COLOR_GREEN)/' | sed 's/:/ $(COLOR_RESET)-/'
	@echo ""

## build: Build the project
build:
	@echo "$(COLOR_BLUE)Building...$(COLOR_RESET)"
	$(GOBUILD) -v ./...

## test: Run all tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	$(GOTEST) -v ./...

## test-short: Run tests in short mode
test-short:
	@echo "$(COLOR_BLUE)Running tests (short mode)...$(COLOR_RESET)"
	$(GOTEST) -short ./...

## test-race: Run tests with race detector
test-race:
	@echo "$(COLOR_BLUE)Running tests with race detector...$(COLOR_RESET)"
	$(GOTEST) -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic `go list ./... | grep -v -e '/examples'`
	@echo "$(COLOR_GREEN)Coverage report generated: $(COVERAGE_FILE)$(COLOR_RESET)"
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(COLOR_GREEN)HTML coverage report: $(COVERAGE_HTML)$(COLOR_RESET)"

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(COLOR_BLUE)Running tests (verbose)...$(COLOR_RESET)"
	$(GOTEST) -v -count=1 ./...

## benchmark: Run benchmarks
benchmark:
	@echo "$(COLOR_BLUE)Running benchmarks...$(COLOR_RESET)"
	$(GOTEST) -bench=. -benchmem ./...

## modernize: Run modernize to apply fixes to all packages
modernize:
	@echo "$(COLOR_BLUE)Running modernize...$(COLOR_RESET)"
	modernize -fix -test ./...

## lint: Run golangci-lint
lint:
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@which golangci-lint > /dev/null || (echo "$(COLOR_YELLOW)golangci-lint not found. Run 'make install-tools'$(COLOR_RESET)" && exit 1)
	golangci-lint run ./...

## fmt: Format all Go files
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	$(GOFMT) -s -w .
	@echo "$(COLOR_GREEN)Code formatted successfully$(COLOR_RESET)"

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(COLOR_BLUE)Checking code formatting...$(COLOR_RESET)"
	@test -z "$$($(GOFMT) -l .)" || (echo "$(COLOR_YELLOW)The following files need formatting:$(COLOR_RESET)" && $(GOFMT) -l . && exit 1)
	@echo "$(COLOR_GREEN)All files are properly formatted$(COLOR_RESET)"

## vet: Run go vet
vet:
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	$(GOVET) ./...

## check: Run fmt-check, vet, and lint
check: fmt-check vet lint
	@echo "$(COLOR_GREEN)All checks passed!$(COLOR_RESET)"

## clean: Clean build artifacts and test cache
clean:
	@echo "$(COLOR_BLUE)Cleaning...$(COLOR_RESET)"
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	rm -rf ./bin
	@echo "$(COLOR_GREEN)Clean complete$(COLOR_RESET)"

## deps: Download dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	$(GOMOD) download
	@echo "$(COLOR_GREEN)Dependencies downloaded$(COLOR_RESET)"

## tidy: Tidy and verify dependencies
tidy:
	@echo "$(COLOR_BLUE)Tidying dependencies...$(COLOR_RESET)"
	$(GOMOD) tidy
	$(GOMOD) verify
	@echo "$(COLOR_GREEN)Dependencies tidied$(COLOR_RESET)"

## install-tools: Install development tools
install-tools:
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "$(COLOR_GREEN)Tools installed$(COLOR_RESET)"

## docs: Generate documentation
docs:
	@echo "$(COLOR_BLUE)Generating documentation...$(COLOR_RESET)"
	@echo "Open http://localhost:6060/pkg/github.com/smallnest/langgraphgo/ in your browser"
	godoc -http=:6060

## examples: Build all examples
examples:
	@echo "$(COLOR_BLUE)Building examples...$(COLOR_RESET)"
	@mkdir -p bin/examples
	@for dir in examples/*/; do \
		example=$$(basename $$dir); \
		echo "  Building $$example..."; \
		$(GOBUILD) -o bin/examples/$$example ./examples/$$example || exit 1; \
	done
	@echo "$(COLOR_GREEN)All examples built successfully$(COLOR_RESET)"

## examples-basic: Build basic examples only
examples-basic:
	@echo "$(COLOR_BLUE)Building basic examples...$(COLOR_RESET)"
	@mkdir -p bin/examples
	@for example in basic_example basic_llm conditional_routing streaming_pipeline; do \
		echo "  Building $$example..."; \
		$(GOBUILD) -o bin/examples/$$example ./examples/$$example || exit 1; \
	done
	@echo "$(COLOR_GREEN)Basic examples built$(COLOR_RESET)"

## ci: Run continuous integration checks
ci: deps check test-race test-coverage
	@echo "$(COLOR_GREEN)CI checks passed!$(COLOR_RESET)"

## pre-commit: Run pre-commit checks (fmt, vet, lint, test)
pre-commit: fmt vet lint test
	@echo "$(COLOR_GREEN)Pre-commit checks passed!$(COLOR_RESET)"

## update-deps: Update all dependencies to latest versions
update-deps:
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "$(COLOR_GREEN)Dependencies updated$(COLOR_RESET)"

## test-checkpoint: Run checkpoint-related tests only
test-checkpoint:
	@echo "$(COLOR_BLUE)Running checkpoint tests...$(COLOR_RESET)"
	$(GOTEST) -v ./checkpoint/...

## test-graph: Run graph-related tests only
test-graph:
	@echo "$(COLOR_BLUE)Running graph tests...$(COLOR_RESET)"
	$(GOTEST) -v ./graph/...

## test-prebuilt: Run prebuilt-related tests only
test-prebuilt:
	@echo "$(COLOR_BLUE)Running prebuilt tests...$(COLOR_RESET)"
	$(GOTEST) -v ./prebuilt/...

## test-ptc: Run PTC-related tests only
test-ptc:
	@echo "$(COLOR_BLUE)Running PTC tests...$(COLOR_RESET)"
	$(GOTEST) -v ./ptc/...

## test-memory: Run memory-related tests only
test-memory:
	@echo "$(COLOR_BLUE)Running memory tests...$(COLOR_RESET)"
	$(GOTEST) -v ./memory/...

## version: Display Go version
version:
	@$(GOCMD) version

## info: Display project information
info:
	@echo "$(COLOR_BOLD)Project Information$(COLOR_RESET)"
	@echo "  Name: LangGraphGo"
	@echo "  Module: github.com/smallnest/langgraphgo"
	@echo "  Go Version: $$(go version | cut -d' ' -f3)"
	@echo "  Packages: $$(find . -name '*.go' -not -path './vendor/*' | xargs dirname | sort -u | wc -l | tr -d ' ')"
	@echo "  Lines of Code: $$(find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1 | awk '{print $$1}')"
	@echo "  Examples: $$(find examples -name 'main.go' | wc -l | tr -d ' ')"
