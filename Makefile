PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
PROTOFILES := $(shell find proto -name '*.proto' | sed -e 's/.proto$$/.pb.go/')
VERSION := $(shell git describe --tags)
VERSION_HASH := $(shell git rev-parse HEAD)

.PHONY: all bench dep build clean test coverage coverhtml lint

all: build

lint: ## Lint the files
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status ${PKG_LIST}

bench: ## Build benchmarking binary (in "$GOPATH/bin"); Run benchmarking
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -v vega/cmd/vegabench
	@go test -run=XXX -bench=. -benchmem -benchtime=1s ./cmd/vegabench

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
	@cat .asciiart.txt
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" vega/cmd/vega
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" vega/cmd/vegabench

proto: ${PROTOFILES} ## build proto definitions

.PRECIOUS: proto/%.pb.go
proto/%.pb.go: proto/%.proto
	@protoc --go_out=. "$<"

cibuild: ## Build the binary file
	@if test -z "$$ARTIFACTS_BIN" ; then echo "No ARTIFACTS_BIN" ; exit 1 ; fi
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -a -i -v -o $(ARTIFACTS_BIN) $(PKG)

clean: ## Remove previous build
	@rm -f ./vega{,bench} ./cmd/{vega/vega,vegabench/vegabench}

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
