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
IMAGE_NAMESPACE?=argoproj-labs
IMAGE_NAME?=${IMAGE_NAMESPACE}/argocd-ephemeral-access

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

##@ Build

.PHONY: build
build: build-ui  ## Build the UI extension

.PHONY: clean-ui
clean-ui: ## delete the extension.tar file.
	find ${UI_DIR} -type f -name extension.tar -delete

.PHONY: build-ui
build-ui: clean-ui ## build the Argo CD UI extension creating the ui/extension.tar file.
	yarn --cwd ${UI_DIR} install
	yarn --cwd ${UI_DIR} build
