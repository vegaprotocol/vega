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

.PHONY: all bench deps build clean grpc grpc_check help test lint mocks proto_check

all: build

lint: ## Lint the files
	@go install golang.org/x/lint/golint
	@golint -set_exit_status ./...

bench: ## Build benchmarking binary (in "$GOPATH/bin"); Run benchmarking
	@go test -run=XXX -bench=. -benchmem -benchtime=1s ./cmd/vegabench

test: deps ## Run unit tests
	@go test -v ./...

race: ## Run data race detector
	@env CGO_ENABLED=1 go test -race ./...

mocks: ## Make mocks
	@if which mockery 1>/dev/null ; then echo "Ignoring mockery found on "'$$PATH'". Using go-run instead." ; fi
	@origdir="$$PWD" ; \
	find . -type d -and -name mocks | while read -r dir ; do \
		cd "$$(dirname "$$dir")" ; \
		go run github.com/vektra/mockery/cmd/mockery -all ; \
		cd "$$origdir" ; \
	done

msan: ## Run memory sanitizer
	@if ! which clang 1>/dev/null ; then echo "Need clang" ; exit 1 ; fi
	@env CC=clang CGO_ENABLED=1 go test -v -msan ./...

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

build: proto ## install the binaries in cmd/{progname}/
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@go build -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -o "./cmd/vega/vega" ./cmd/vega
	@go build -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -o "./cmd/vegabench/vegabench" ./cmd/vegabench

install: proto ## install the binary in GOPATH/bin
	@cat .asciiart.txt
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" ./cmd/vega
	@go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" ./cmd/vegabench

gqlgen: deps ## run gqlgen
	@cd ./internal/api/endpoints/gql && go run github.com/99designs/gqlgen -c gqlgen.yml

proto: ${PROTOFILES} ## build proto definitions

.PRECIOUS: proto/%.pb.go
proto/%.pb.go: proto/%.proto
	@protoc --go_out=. "$<"

proto_check: ## proto: Check committed files match just-generated files
	@touch proto/*.proto ; \
	make proto 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only proto/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff proto/ ; \
		exit 1 ; \
	fi

# GRPC Targets

grpc: internal/api/grpc.swagger.json internal/api/grpc.pb.gw.go internal/api/grpc.pb.go ## Generate gRPC files: grpc.swagger.json, grpc.pb.gw.go, grpc.pb.go

internal/api/grpc.pb.go: internal/api/grpc.proto
	@protoc -I. -Iinternal/api/ --go_out=plugins=grpc:. "$<" && \
	sed --in-place -re 's/proto1 "proto"/proto1 "code.vegaprotocol.io\/vega\/proto"/' "$@"

GRPC_CONF_OPT := logtostderr=true,grpc_api_configuration=internal/api/grpc-rest-bindings.yml:.

# This creates a reverse proxy to forward HTTP requests into gRPC requests
internal/api/grpc.pb.gw.go: internal/api/grpc.proto internal/api/grpc-rest-bindings.yml
	@protoc -I. -Iinternal/api/ --grpc-gateway_out=$(GRPC_CONF_OPT) "$<" && \
	sed --in-place -re 's/proto_0 "proto"/proto_0 "code.vegaprotocol.io\/vega\/proto"/' "$@"

# Generate Swagger documentation
internal/api/grpc.swagger.json: internal/api/grpc.proto internal/api/grpc-rest-bindings.yml
	@protoc -I. -Iinternal/api/ --swagger_out=$(GRPC_CONF_OPT) "$<"

grpc_check: ## gRPC: Check committed files match just-generated files
	@touch internal/api/*.proto ; \
	make grpc 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only internal/api/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff internal/api/ ; \
		exit 1 ; \
	fi

# Misc Targets

clean: ## Remove previous build
	@rm -f ./vega{,bench} ./cmd/{vega/vega,vegabench/vegabench}

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
