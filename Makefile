PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

ifndef ARTIFACTS_BIN ## environment variable override from gitlab-ci
$(info $$ARTIFACTS_BIN is [${ARTIFACTS_BIN}])
ARTIFACTS_BIN := "./$(PROJECT_NAME)"
endif

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status ${PKG_LIST}

test: deps ## Run unit tests
	@go test -short ${PKG_LIST} -v

race: ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

coverage: ## Generate global code coverage report
	./coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./coverage.sh html;

deps: ## Get the dependencies
	@go mod download

install: proto ## install the binary in GOPATH/bin
	@go install -v vega/cmd/vega

proto: ## build proto definitions
	@protoc --go_out=. ./proto/*.proto

build: ## Build the binary file
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -i -v -o $(ARTIFACTS_BIN) $(PKG)

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

.PHONY: proto

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
