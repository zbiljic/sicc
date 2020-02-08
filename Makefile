SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
THIS_OS := $(shell uname)

# Using directory as project name.
PROJECT_NAME := $(shell basename $(PROJECT_ROOT))

GO_LDFLAGS := "-s -w"

default: help

ifeq (,$(findstring $(THIS_OS),Darwin Linux FreeBSD Windows))
$(error Building is currently only supported on Darwin and Linux.)
endif

ALL_TARGETS += linux_amd64 \
	windows_amd64

# On MacOS, we only build for MacOS
ifeq (Darwin,$(THIS_OS))
ALL_TARGETS += darwin_amd64
endif

dist/darwin_amd64/$(PROJECT_NAME):
	@echo "==> Building $@ ..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		go build \
		-trimpath \
		-ldflags $(GO_LDFLAGS) \
		-o "$@"

dist/linux_amd64/$(PROJECT_NAME):
	@echo "==> Building $@ ..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build \
		-trimpath \
		-ldflags $(GO_LDFLAGS) \
		-o "$@"

dist/windows_amd64/$(PROJECT_NAME):
	@echo "==> Building $@ ..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build \
		-trimpath \
		-ldflags $(GO_LDFLAGS) \
		-o "$@.exe"

# Define package targets for each of the build targets we actually have on this system
define makePackageTarget

dist/$(1).zip: dist/$(1)/$(PROJECT_NAME)
	@echo "==> Packaging for $(1)..."
	@zip -j dist/$(1).zip dist/$(1)/*
	@cat dist/$(1).zip | sha256sum > dist/$(1).zip.sha256
	@truncate -s 64 dist/$(1).zip.sha256

endef

# Reify the package targets
$(foreach t,$(ALL_TARGETS),$(eval $(call makePackageTarget,$(t))))

# Only for CI compliance
.PHONY: bootstrap
bootstrap: lint-deps # Install all dependencies

.PHONY: lint-deps
lint-deps: ## Install linter dependencies
	@echo "==> Updating linter dependencies..."
	@which golangci-lint 2>/dev/null || go get -u github.com/golangci/golangci-lint/cmd/golangci-lint && echo "Installed golangci-lint"

.PHONY: check
check: ## Lint the source code
	@echo "==> Linting source code..."
	@golangci-lint run \
		--no-config \
		--issues-exit-code=0 \
		--deadline=10m \
		--disable-all \
		--enable=govet \
		--enable=errcheck \
		--enable=staticcheck \
		--enable=unused \
		--enable=gosimple \
		--enable=structcheck \
		--enable=varcheck \
		--enable=ineffassign \
		--enable=deadcode \
		--enable=typecheck \
		--enable=bodyclose \
		--enable=golint \
		--enable=stylecheck \
		--enable=gosec \
		--enable=interfacer \
		--enable=unconvert \
		--enable=dupl \
		--enable=goconst \
		--enable=gocyclo \
		--enable=gocognit \
		--enable=gofmt \
		--enable=goimports \
		--enable=maligned \
		--enable=misspell \
		--enable=lll \
		--enable=unparam \
		--enable=dogsled \
		--enable=nakedret \
		--enable=prealloc \
		--enable=scopelint \
		--enable=gocritic \
		--enable=godox \
		--enable=funlen \
		--enable=whitespace \
		--enable=wsl \
		--enable=gomnd \
		./...

.PHONY: test
test: LOCAL_PACKAGES = $(shell go list ./... | grep -v '/vendor/')
test: ## Run the test suite and/or any other tests
	@echo "==> Running test suites..."
	@go test \
		-v \
		-cover \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-timeout=900s \
		$(LOCAL_PACKAGES)

.PHONY: coverage
coverage: ## Open a web browser displaying coverage
	@go tool cover -html=coverage.txt

.PHONY: build
build: clean $(foreach t,$(ALL_TARGETS),dist/$(t).zip) ## Build release packages
	@echo "==> Results:"
	@tree --dirsfirst $(PROJECT_ROOT)/dist

.PHONY: clean
clean: ## Remove build artifacts
	@echo "==> Cleaning build artifacts..."
	@rm -fv coverage.txt
	@find . -name '*.test' | xargs rm -fv
	@rm -rfv "$(PROJECT_ROOT)/dist/"

HELP_FORMAT="    \033[36m%-15s\033[0m %s\n"
.PHONY: help
help: ## Display this usage information
	@echo "Valid targets:"
	@grep -E '^[^ ]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo

FORCE:
