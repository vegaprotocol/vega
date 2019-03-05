PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
PROTOFILES := $(shell find proto -name '*.proto' | sed -e 's/.proto$$/.pb.go/')
TAG := $(shell git describe --tags 2>/dev/null)

# See https://docs.gitlab.com/ce/ci/variables/README.html for CI vars.
ifeq ($(CI),)
	# Not in CI
	ifeq ($(TAG),)
		# No tag, so make one
		VERSION := dev-$(USER)
	else
		VERSION := dev-$(TAG)
	endif
	VERSION_HASH := $(shell git rev-parse HEAD | cut -b1-8)
else
	# In CI
	ifeq ($(TAG),)
		# No tag, so make one
		VERSION := interim-$(CI_COMMIT_REF_SLUG)
	else
		VERSION := $(TAG)
	endif
	VERSION_HASH := $(CI_COMMIT_SHORT_SHA)
endif

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
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" ./cmd/vega
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" ./cmd/vegabench

gqlgen: deps ## run gqlgen
	@cd ./api/endpoints/gql && go run github.com/99designs/gqlgen -c gqlgen.yml


proto: ${PROTOFILES} ## build proto definitions

.PRECIOUS: proto/%.pb.go
proto/%.pb.go: proto/%.proto
	@protoc --go_out=. "$<"

clean: ## Remove previous build
	@rm -f ./vega{,bench} ./cmd/{vega/vega,vegabench/vegabench}

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
