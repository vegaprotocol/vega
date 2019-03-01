PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
PROTOFILES := $(shell find proto -name '*.proto' | sed -e 's/.proto$$/.pb.go/')
VERSION := $(shell git describe --tags)
VERSION_HASH := $(shell git rev-parse HEAD)

.PHONY: all bench deps build clean test lint

all: build

lint: ## Lint the files
	@go get -u golang.org/x/lint/golint
	@golint -set_exit_status ./...

bench: ## Build benchmarking binary (in "$GOPATH/bin"); Run benchmarking
	@go test -run=XXX -bench=. -benchmem -benchtime=1s ./cmd/vegabench

test: deps ## Run unit tests
	@go test ./...

race: ## Run data race detector
	@go test -race ./...

msan: ## Run memory sanitizer
	@if ! which clang 1>/dev/null ; then echo "Need clang" ; exit 1 ; fi
	@env CC=clang go test -msan ./...

.PHONY: .testCoverage.txt
.testCoverage.txt:
	@go test -covermode=count -coverprofile="$@" ./...
	@go tool cover -func="$@"

coverage: .testCoverage.txt ## Generate global code coverage report

.PHONY: .testCoverage.html
.testCoverage.html: .testCoverage.txt
	@go tool cover -html="$^" -o "$@"

coveragehtml: .testCoverage.html ## Generate global code coverage report in HTML

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
