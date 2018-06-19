PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST} -v -coverprofile .testCoverage.txt

race: dep ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

coverage: ## Generate global code coverage report
	./coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./coverage.sh html;

dep: ## Get the dependencies
	@dep ensure

build: dep ## Build the binary file
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -i -v $(PKG)

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
