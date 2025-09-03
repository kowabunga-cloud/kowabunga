# Copyright (c) The Kowabunga Project
# Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
# SPDX-License-Identifier: Apache-2.0

PKG_NAME=github.com/kowabunga-cloud/kowabunga/kowabunga/kahuna
VERSION=0.63.2
DIST=noble
CODENAME=NoFuture

SRC_DIR = kowabunga
SDK_GENERATOR = go-server
SDK_PACKAGE_NAME = sdk
SDK_VERSION = "tags/v0.52.5"
#SDK_VERSION = "heads/main"
SDK_OPENAPI_SPEC = "https://raw.githubusercontent.com/kowabunga-cloud/openapi/refs/$(SDK_VERSION)/openapi.yaml"

#export GOOS=linux
#export GOARCH=amd64

# Make sure GOPATH is NOT set to this folder or we'll get an error "$GOPATH/go.mod exists but should not"
#export GOPATH = ""
export GO111MODULE = on
BINDIR = bin
PLUGINS_DIR = plugins
PLUGINS_KAKTUS_PKG_DIR = ./kowabunga/kaktus/plugins

NODE_DIR = ./node_modules
YARN = $(NODE_DIR)/.bin/yarn
OPENAPI_GENERATOR = $(NODE_DIR)/.bin/openapi-generator-cli

GOLINT = $(BINDIR)/golangci-lint
GOLINT_VERSION = v2.4.0

GOVULNCHECK = $(BINDIR)/govulncheck
GOVULNCHECK_VERSION = v1.1.4

GOSEC = $(BINDIR)/gosec
GOSEC_VERSION = v2.22.8

PKGS = $(shell go list ./... | grep -v /$(SDK_PACKAGE_NAME))
PKGS_SHORT = $(shell go list ./... | grep -v /$(SDK_PACKAGE_NAME) | sed 's%github.com/kowabunga-cloud/kowabunga/%%')

V = 0
Q = $(if $(filter 1,$V),,@)
PROD = 0
ifeq ($(PROD),1)
DEBUG = -w -s
endif
M = $(shell printf "\033[34;1m▶\033[0m")

ifeq ($(V), 1)
  OUT = ""
else
  OUT = ">/dev/null"
endif

UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
.EXPORT_ALL_VARIABLES:
OSX_CEPH_DIR = $(shell ls -d /opt/homebrew/Cellar/ceph-client/* | tail -n 1)
CGO_CPPFLAGS = "-I$(OSX_CEPH_DIR)/include"
CGO_LDFLAGS = "-L$(OSX_CEPH_DIR)/lib"
endif

# This is our default target
# it does not build/run the tests
.PHONY: all
all: mod fmt vet lint build ; @ ## Do all
	$Q echo "done"

.PHONY: get-yarn
get-yarn: bin ;$(info $(M) [NPM] installing yarn…) @
	$Q test -x $(YARN) || npm install --silent yarn

.PHONY: get-openapi-generator
get-openapi-generator: get-yarn ;$(info $(M) [Yarn] installing openapi-generator-cli…) @
	$Q test -x $(OPENAPI_GENERATOR) || $(YARN) add --silent @openapitools/openapi-generator-cli 2> /dev/null
	$Q chmod a+x $(OPENAPI_GENERATOR)

# Generates server-side SDK from OpenAPI specification
.PHONY: sdk
sdk: get-openapi-generator ; $(info $(M) generate server-side SDK from OpenAPI specifications…) @
	$Q git rm -qrf $(SRC_DIR)/$(SDK_PACKAGE_NAME) || true
	$Q $(OPENAPI_GENERATOR) generate \
	  -g $(SDK_GENERATOR) \
	  --package-name $(SDK_PACKAGE_NAME) \
	  --openapi-normalizer KEEP_ONLY_FIRST_TAG_IN_OPERATION=true \
	  -p outputAsLibrary=true \
	  -p sourceFolder=$(SDK_PACKAGE_NAME) \
	  -i "$(SDK_OPENAPI_SPEC)" \
	  -o $(SRC_DIR) \
	  $(OUT)
	$Q rm -f $(SRC_DIR)/README.md
	$Q rm -f $(SRC_DIR)/.openapi-generator-ignore
	$Q rm -rf $(SRC_DIR)/.openapi-generator
	$Q rm -rf $(SRC_DIR)/api
	$Q git add "$(SRC_DIR)/$(SDK_PACKAGE_NAME)"

# This target grabs the necessary go modules
.PHONY: mod
mod: ; $(info $(M) collecting modules…) @
	$Q go mod download
	$Q go mod tidy

# Updates all go modules
update: ; $(info $(M) updating modules…) @
	$Q go get -u ./...
	$Q go mod tidy

# Makes sure bin directory is created
.PHONY: bin
bin: ; $(info $(M) create local bin directory) @
	$Q mkdir -p $(BINDIR)

.PHONY: kahuna
kahuna: bin ; $(info $(M) building Kahuna orchestrator…) @
	$Q go build \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG) -X $(PKG_NAME).version=$(VERSION) -X $(PKG_NAME).codename=$(CODENAME)' \
		-o $(BINDIR) ./cmd/kahuna

.PHONY: kaktus
kaktus: ; $(info $(M) building Kaktus agent…) @
	$Q go build \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG)' \
		-o $(BINDIR) ./cmd/kaktus

.PHONY: kawaii
kawaii: ; $(info $(M) building Kawaii agent…) @
	$Q go build \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG)' \
		-o $(BINDIR) ./cmd/kawaii

.PHONY: kiwi
kiwi: ; $(info $(M) building Kiwi agent…) @
	$Q go build \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG)' \
		-o $(BINDIR) ./cmd/kiwi

.PHONY: konvey
konvey: ; $(info $(M) building Konvey agent…) @
	$Q go build \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG)' \
		-o $(BINDIR) ./cmd/konvey

.PHONY: kowarp
kowarp: ; $(info $(M) building Kowarp agent…) @
	$Q go build \
                -gcflags="kowabunga/...=-e" \
                -ldflags='$(DEBUG)' \
                -o $(BINDIR) ./cmd/kowarp

# Makes sure plugins directory is created
.PHONY: plugins
plugins: ; $(info $(M) create local plugins directory) @
	$Q mkdir -p $(PLUGINS_DIR)

.PHONY: plugin-ceph
plugin-ceph: plugins ; $(info $(M) building Kaktus Ceph plugin…) @
	$Q go build -buildmode=plugin \
		-gcflags="kowabunga/...=-e" \
		-ldflags='$(DEBUG)' \
		-o $(PLUGINS_DIR) $(PLUGINS_KAKTUS_PKG_DIR)/ceph

.PHONY: build
build: kahuna kaktus kawaii kiwi konvey plugin-ceph

.PHONY: tests
tests: ; $(info $(M) testing Kowabunga suite…) @
	$Q go test ./... -count=1 -coverprofile=coverage.txt

.PHONY: deb
deb: ; $(info $(M) building debian package…) @ ## Build debian package
	$Q VERSION=$(VERSION) DIST=$(DIST) ./debian.sh

.PHONY: get-lint
get-lint: ; $(info $(M) downloading go-lint…) @
	$Q test -x $(GOLINT) || sh -c $(GOLINT) --version 2> /dev/null| grep $(GOLINT_VERSION)  || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLINT_VERSION)

.PHONY: lint
lint: get-lint ; $(info $(M) running go-lint…) @
	$Q $(GOLINT) run ./... ; exit 0

.PHONY: get-govulncheck
get-govulncheck: ; $(info $(M) downloading govulncheck…) @
	$Q test -x $(GOVULNCHECK) || GOBIN="$(PWD)/$(BINDIR)/" go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: vuln
vuln: get-govulncheck ; $(info $(M) running govulncheck…) @ ## Check for known vulnerabilities
	$Q $(GOVULNCHECK) ./... ; exit 0

.PHONY: get-gosec
get-gosec: ; $(info $(M) downloading gosec…) @
	$Q test -x $(GOSEC) || GOBIN="$(PWD)/$(BINDIR)/" go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

.PHONY: sec
sec: get-gosec ; $(info $(M) running gosec…) @ ## AST / SSA code checks
	$Q $(GOSEC) -terse -exclude=G101,G115 ./... ; exit 0

.PHONY: vet
vet: ; $(info $(M) running go vet…) @
	$Q go vet $(PKGS) ; exit 0

.PHONY: fmt
fmt: ; $(info $(M) running go fmt…) @
	$Q gofmt -w -s $(PKGS_SHORT)

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	$Q rm -rf $(BINDIR)
	$Q rm -rf $(PLUGINS_DIR)
	$Q rm -rf $(NODE_DIR)
	$Q rm -f package-lock.json
	$Q rm -f package.json
	$Q rm -f yarn.lock
	$Q rm -f openapitools.json

# This target parse this makefile and extract special comments to build a help
.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# This target count all the lines of .go files (no matter if empty lines or comments)
.PHONY: lc
lc: ; @ ## Count lines
	@find . -name "*.go" -exec cat {} \; | wc -l | awk '{print $$1}'

# This target count the lines of go code only (ignore empty lines, comments, etc.)
# it requires gosloc
.PHONY: sloc
sloc: ; @ ## Count GO lines
	@find . -name "*.go" -exec cat {} \; | gosloc

# This target print the version to be used as version if build is launched
# this file does not exists in our VCS but Jenkins create the file before building the project
.PHONY: version
version:
	@echo $(VERSION)
