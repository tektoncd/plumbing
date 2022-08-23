MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
BIN      = $(CURDIR)/.bin

GOLANGCI_VERSION = v1.47.2

GO           = go
TIMEOUT_UNIT = 5m
TIMEOUT_E2E  = 20m
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

export GO111MODULE=on

COMMANDS=$(patsubst cmd/%,%,$(wildcard cmd/*))
BINARIES=$(addprefix bin/,$(COMMANDS))

.PHONY: all
all: fmt $(BINARIES) | $(BIN) ; $(info $(M) building executable…) @ ## Build program binary

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

FORCE:

bin/%: cmd/% FORCE
	$Q $(GO) build $(LDFLAGS) -v -o $@ ./$<

.PHONY: cross
cross: amd64 arm arm64 s390x ppc64le ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) ./cmd/...

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build $(LDFLAGS) ./cmd/...

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) ./cmd/...

.PHONY: s390x
s390x:
	GOOS=linux GOARCH=s390x go build $(LDFLAGS) ./cmd/...

.PHONY: ppc64le
ppc64le:
	GOOS=linux GOARCH=ppc64le go build $(LDFLAGS) ./cmd/...

KO = $(or ${KO_BIN},${KO_BIN},$(BIN)/ko)
$(BIN)/ko: PACKAGE=github.com/google/ko

.PHONY: apply
apply: | $(KO) ; $(info $(M) ko apply -R -f config/) @ ## Apply config to the current cluster
	$Q $(KO) apply -R -f config

.PHONY: resolve
resolve: | $(KO) ; $(info $(M) ko resolve -R -f config/) @ ## Resolve config to the current cluster
	$Q $(KO) resolve --push=false --oci-layout-path=$(BIN)/oci -R -f config

## Tests
TEST_UNIT_TARGETS := test-unit-verbose test-unit-race
test-unit-verbose: ARGS=-v
test-unit-race:    ARGS=-race
$(TEST_UNIT_TARGETS): test-unit
.PHONY: $(TEST_UNIT_TARGETS) test-unit
test-unit: ## Run unit tests
	$(GO) test -timeout $(TIMEOUT_UNIT) $(ARGS) ./...

TEST_E2E_TARGETS := test-e2e-short test-e2e-verbose test-e2e-race
test-e2e-short:   ARGS=-short
test-e2e-verbose: ARGS=-v
test-e2e-race:    ARGS=-race
$(TEST_E2E_TARGETS): test-e2e
.PHONY: $(TEST_E2E_TARGETS) test-e2e
test-e2e:  ## Run end-to-end tests
	$(GO) test -timeout $(TIMEOUT_E2E) -tags e2e $(ARGS) ./test/...

.PHONY: check tests
check tests: test-unit test-e2e
## Linters configuration and targets
# TODO(vdemeester) gofmt and goimports checks (run them with -w and make a diff)

GOLINT = $(BIN)/golint
$(BIN)/golint: PACKAGE=golang.org/x/lint/golint

.PHONY: golint
golint: | $(GOLINT) ; $(info $(M) running golint…) @ ## Run golint
	$Q $(GOLINT) -set_exit_status ./...

.PHONY: vet
vet: | ; $(info $(M) running go vet…) @ ## Run go vet
	$Q go vet ./...

GOLANGCILINT = $(BIN)/golangci-lint
$(BIN)/golangci-lint: ; $(info $(M) getting golangci-lint $(GOLANGCI_VERSION))
	cd tools; GOBIN=$(BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

.PHONY: golangci-lint
golangci-lint: | $(GOLANGCILINT) ; $(info $(M) running golangci-lint…) @ ## Run golangci-lint
	$Q $(GOLANGCILINT) run --max-issues-per-linter=0 --max-same-issues=0 --deadline 5m

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: PACKAGE=golang.org/x/tools/cmd/goimports

.PHONY: goimports
goimports: | $(GOIMPORTS) ; $(info $(M) running goimports…) ## Run goimports
	$Q $(GOIMPORTS) -l -e -w pkg cmd

.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	$Q $(GO) fmt ./...

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'
