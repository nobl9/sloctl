.DEFAULT_GOAL := help
MAKEFLAGS += --silent --no-print-directory

BIN_DIR := ./bin
TEST_DIR := ./test
APP_NAME := sloctl
LDFLAGS += -s -w
VERSION_PKG := "$(shell go list -m)/internal"

ifndef BRANCH
  BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
endif
ifndef REVISION
  REVISION := $(shell git rev-parse --short=8 HEAD)
endif

# renovate datasource=github-releases depName=securego/gosec
GOSEC_VERSION := v2.21.4
# renovate datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION := v1.61.0
# renovate datasource=go depName=golang.org/x/vuln/cmd/govulncheck
GOVULNCHECK_VERSION := v1.1.3
# renovate datasource=go depName=golang.org/x/tools/cmd/goimports
GOIMPORTS_VERSION := v0.25.0

# Check if the program is present in $PATH and install otherwise.
# ${1} - oneOf{binary,yarn}
# ${2} - program name
define _ensure_installed
	LOCAL_BIN_DIR=$(BIN_DIR) ./scripts/ensure_installed.sh "${1}" "${2}"
endef

# Install Go binary using 'go install' with an output directory set via $GOBIN.
# ${1} - repository url
define _install_go_binary
	GOBIN=$(realpath $(BIN_DIR)) go install "${1}"
endef

# Print Makefile target step description for check.
# Only print top level steps this way, and not dependent steps, like 'install'.
# ${1} - step description
define _print_step
	printf -- '------\n%s...\n' "${1}"
endef

# Build sloctl docker image.
# ${1} - image name
# ${2} - version
# ${3} - git branch
# ${4} - git revision
define _build_docker
	docker build \
		--build-arg LDFLAGS="-X $(VERSION_PKG).BuildVersion=$(2) -X $(VERSION_PKG).BuildGitBranch=$(3) -X $(VERSION_PKG).BuildGitRevision=$(4)" \
		-t "$(1)" .
endef

.PHONY: build
## Build sloctl binary.
build:
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)/

.PHONY: docker
## Build sloctl Docker image.
docker:
	$(call _build_docker,sloctl,$(VERSION),$(BRANCH),$(REVISION))

.PHONY: test/unit test/go/unit test/bats/%
## Run all unit tests.
test/unit: test/go/unit test/bats/unit

.PHONY: test/e2e test/bats/unit test/bats/e2e test/go/e2e-docker
## Run all e2e tests.
test/e2e: test/bats/e2e test/go/e2e-docker

## Run go unit tests.
test/go/unit:
	$(call _print_step,Running go unit tests)
	go test -race -cover ./...

## Run go e2e docker tests.
test/go/e2e-docker:
	$(call _print_step,Running go docker image tests)
	go test -race -tags=e2e_test ./...

## Run bats unit tests.
test/bats/unit:
	$(call _print_step,Running bats unit tests)
	$(call _build_docker,sloctl-unit-test-bin,v1.0.0,PC-123-test,e2602ddc)
	docker build -t sloctl-bats-unit -f $(TEST_DIR)/docker/Dockerfile.unit .
	docker run -e TERM=linux --rm \
		sloctl-bats-unit -F pretty --filter-tags unit $(TEST_DIR)/*

## Run bats e2e tests.
test/bats/e2e:
	$(call _print_step,Running bats e2e tests)
	$(call _build_docker,sloctl-e2e-test-bin,$(VERSION),$(BRANCH),$(REVISION))
	docker build -t sloctl-bats-e2e -f $(TEST_DIR)/docker/Dockerfile.e2e .
	docker run --rm \
		-e SLOCTL_URL=$(SLOCTL_URL) \
		-e SLOCTL_CLIENT_ID=$(SLOCTL_CLIENT_ID) \
		-e SLOCTL_CLIENT_SECRET=$(SLOCTL_CLIENT_SECRET) \
		-e SLOCTL_OKTA_ORG_URL=$(SLOCTL_OKTA_ORG_URL) \
		-e SLOCTL_OKTA_AUTH_SERVER=$(SLOCTL_OKTA_AUTH_SERVER) \
		-e SLOCTL_GIT_REVISION=$(REVISION) \
		sloctl-bats-e2e -F pretty --filter-tags e2e $(TEST_DIR)/*

.PHONY: check check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns
## Run all checks.
check: check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns

## Run 'go vet' on the whole project.
check/vet:
	$(call _print_step,Running go vet)
	go vet ./...

## Run golangci-lint all-in-one linter with configuration defined inside .golangci.yml.
check/lint:
	$(call _print_step,Running golangci-lint)
	$(call _ensure_installed,binary,golangci-lint)
	$(BIN_DIR)/golangci-lint run

## Check for security problems using gosec, which inspects the Go code by scanning the AST.
check/gosec:
	$(call _print_step,Running gosec)
	$(call _ensure_installed,binary,gosec)
	$(BIN_DIR)/gosec -exclude-generated -quiet ./...

## Check spelling, rules are defined in cspell.json.
check/spell:
	$(call _print_step,Verifying spelling)
	$(call _ensure_installed,yarn,cspell)
	yarn --silent cspell --no-progress '**/**'

## Check for trailing whitespaces in any of the projects' files.
check/trailing:
	$(call _print_step,Looking for trailing whitespaces)
	yarn --silent check-trailing-whitespaces

## Check markdown files for potential issues with markdownlint.
check/markdown:
	$(call _print_step,Verifying Markdown files)
	$(call _ensure_installed,yarn,markdownlint)
	yarn --silent markdownlint '**/*.md' --ignore node_modules

## Check for potential vulnerabilities across all Go dependencies.
check/vulns:
	$(call _print_step,Running govulncheck)
	$(call _ensure_installed,binary,govulncheck)
	$(BIN_DIR)/govulncheck ./...

## Verify if the auto generated code has been committed.
check/generate:
	$(call _print_step,Checking if generated code matches the provided definitions)
	./scripts/check-generate.sh

## Verify if the files are formatted.
## You must first commit the changes, otherwise it won't detect the diffs.
check/format:
	$(call _print_step,Checking if files are formatted)
	./scripts/check-formatting.sh

.PHONY: generate generate/code
## Auto generate files.
generate: generate/code

## Generate Golang code.
generate/code:
	echo "Generating Go code..."
	go generate ./...

.PHONY: format format/go format/cspell
## Format files.
format: format/go format/cspell

## Format Go files.
format/go:
	echo "Formatting Go files..."
	$(call _ensure_installed,binary,goimports)
	gofmt -w -l -s .
	$(BIN_DIR)/goimports -local=github.com/nobl9/sloctl -w .

## Format cspell config file.
format/cspell:
	echo "Formatting cspell.yaml configuration (words list)..."
	$(call _ensure_installed,yarn,yaml)
	yarn --silent format-cspell-config

.PHONY: install install/yarn install/golangci-lint install/gosec install/govulncheck install/goimports
## Install all dev dependencies.
install: install/yarn install/golangci-lint install/gosec install/govulncheck install/goimports

## Install JS dependencies with yarn.
install/yarn:
	echo "Installing yarn dependencies..."
	yarn --silent install

## Install golangci-lint (https://golangci-lint.run).
install/golangci-lint:
	echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |\
 		sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

## Install gosec (https://github.com/securego/gosec).
install/gosec:
	echo "Installing gosec..."
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |\
 		sh -s -- -b $(BIN_DIR) $(GOSEC_VERSION)

## Install govulncheck (https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
install/govulncheck:
	echo "Installing govulncheck..."
	$(call _install_go_binary,golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION))

## Install goimports (https://pkg.go.dev/golang.org/x/tools/cmd/goimports).
install/goimports:
	echo "Installing goimports..."
	$(call _install_go_binary,golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION))

.PHONY: help
## Print this help message.
help:
	./scripts/makefile-help.awk $(MAKEFILE_LIST)
