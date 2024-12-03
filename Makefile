CURRENT_DIR=$(shell pwd)
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GIT_TAG:=$(if $(GIT_TAG),$(GIT_TAG),$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi))

# docker image publishing options
IMAGE_NAMESPACE?=us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images
IMAGE_NAME?=${IMAGE_NAMESPACE}/reference-api

ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
else
IMAGE_TAG?=latest
endif

IMG=${IMAGE_NAME}:${IMAGE_TAG}

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

UI_DIR=${CURRENT_DIR}/ui

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	cd reference-api && go fmt .

.PHONY: vet
vet: ## Run go vet against code.
	cd reference-api && go vet .

##@ Lint
.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	cd reference-api && $(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	cd reference-api && $(GOLANGCI_LINT) run --fix


##@ Build

.PHONY: build
build: build-ui build-go ## Build the UI extension and the reference-api binary

.PHONY: build-go
build-go: fmt vet ## Build the ephemeral-access binary.
	cd reference-api && go build -o bin/reference-api .

.PHONY: clean-ui
clean-ui: ## delete the extension.tar file.
	find ${UI_DIR} -type f -name extension.tar -delete

.PHONY: build-ui
build-ui: clean-ui ## build the Argo CD UI extension creating the ui/extension.tar file.
	yarn --cwd ${UI_DIR} install
	yarn --cwd ${UI_DIR} build

.PHONY: goreleaser-build-local
goreleaser-build-local: goreleaser ## Run goreleaser build locally. Use to validate the goreleaser configuration.
	$(GORELEASER) build --snapshot --clean --single-target --verbose

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
MOCKERY ?= $(LOCALBIN)/mockery-$(MOCKERY_VERSION)
GORELEASER ?= $(LOCALBIN)/goreleaser-$(GORELEASER_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
GOLANGCI_LINT_VERSION ?= v1.57.2
MOCKERY_VERSION ?= v2.45.0
GORELEASER_VERSION ?= v2.3.2

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download goreleaser locally if necessary.
$(GORELEASER): $(LOCALBIN)
	$(call go-install-tool,$(GORELEASER),github.com/goreleaser/goreleaser/v2,$(GORELEASER_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef