BINARY_NAME = tepautomation

GO  = go
BIN = $(CURDIR)/.bin

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

all: bin/$(BINARY_NAME) test

FORCE:

.PHONY: cross
cross: amd64 386 arm arm64 s390x ppc64le ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	GOOS=windows GOARCH=amd64 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64 ./cmd/$(BINARY_NAME)
	GOOS=darwin GOARCH=amd64 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)

.PHONY: 386
386:
	GOOS=linux GOARCH=386 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-386 ./cmd/$(BINARY_NAME)
	GOOS=windows GOARCH=386 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-386 ./cmd/$(BINARY_NAME)

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm ./cmd/$(BINARY_NAME)

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)

.PHONY: s390x
s390x:
	GOOS=linux GOARCH=s390x go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-s390x ./cmd/$(BINARY_NAME)

.PHONY: ppc64le
ppc64le:
	GOOS=linux GOARCH=ppc64le go build  $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-ppc64le ./cmd/$(BINARY_NAME)

bin/%: cmd/% FORCE
	go build  $(LDFLAGS) -v -o $@ ./$<

check: lint test

.PHONY: test
test: test-unit ## run all tests

.PHONY: test-unit
test-unit: ## run unit tests
	@echo "Running unit tests..."
	@$(GO) test -short -cover ./...

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {gsub("\\\\n",sprintf("\n%22c",""), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

GOLANGCILINT = $(BIN)/golangci-lint
$(BIN)/golangci-lint:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN) $(GOLANGCI_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCILINT)
	@echo "Running golangci-lint..."
	@$(GOLANGCILINT) run --max-issues-per-linter=0 --max-same-issues=0 --deadline 5m

.PHONY: lint
lint: golangci-lint

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: PACKAGE=golang.org/x/tools/cmd/goimports

.PHONY: goimports
goimports: $(GOIMPORTS)
	@echo "Running goimports..."
	@$(GOIMPORTS) -l -e -w pkg cmd

.PHONY: fmt
fmt: goimports
	@echo "Running go fmt..."
	@$(GO) fmt `go list ./... | grep -v /vendor/`
