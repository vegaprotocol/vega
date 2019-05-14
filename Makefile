PROJECT_NAME := "vega"
PKG := "./cmd/$(PROJECT_NAME)"
PROTOFILES := $(shell find proto -name '*.proto' | sed -e 's/.proto$$/.pb.go/')
PROTOVALFILES := $(shell find proto -name '*.proto' | sed -e 's/.proto$$/.validator.pb.go/')
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

.PHONY: all bench deps build clean docker docker_quick gettools grpc grpc_check help test lint mocks proto_check

all: build

lint: ## Lint the files
	@go install golang.org/x/lint/golint
	@go list ./... | xargs -r golint -set_exit_status | sed -e "s#^$$GOPATH/src/##"

bench: ## Build benchmarking binary (in "$GOPATH/bin"); Run benchmarking
	@go test -run=XXX -bench=. -benchmem -benchtime=1s ./cmd/vegabench

test: deps ## Run unit tests
	@go test ./...

race: ## Run data race detector
	@env CGO_ENABLED=1 TEST_SKIP_RACE=yes go test -race ./...

mocks: ## Make mocks
	@go generate ./internal/...

msan: ## Run memory sanitizer
	@if ! which clang 1>/dev/null ; then echo "Need clang" ; exit 1 ; fi
	@env CC=clang CGO_ENABLED=1 go test -msan ./...

vet: ## Run go vet
	@go vet -all ./...

.PRECIOUS: .testCoverage.txt
.testCoverage.txt:
	@go test -covermode=count -coverprofile="$@" ./...
	@go tool cover -func="$@"

coverage: .testCoverage.txt ## Generate global code coverage report

.PRECIOUS: .testCoverage.html
.testCoverage.html: .testCoverage.txt
	@go tool cover -html="$^" -o "$@"

coveragehtml: .testCoverage.html ## Generate global code coverage report in HTML

deps: ## Get the dependencies
	@go mod download
	@which modvendor 1>/dev/null || go get github.com/vegaprotocol/modvendor@v0.0.1
	@which protoc-gen-govalidators 1>/dev/null || go get github.com/mwitkow/go-proto-validators/protoc-gen-govalidators@v0.0.0-20190212092829-1f388280e944
	@go mod vendor
	@grep 'google/protobuf' go.mod | awk '{print "# " $$1 " " $$2 "\n"$$1"/src";}' >> vendor/modules.txt
	@modvendor -copy="**/*.proto"

build: proto ## install the binaries in cmd/{progname}/
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@for app in vega vegabench ; do \
		env CGO_ENABLED=0 go build -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -o "./cmd/$$app/$$app" "./cmd/$$app" || exit 1 ; \
	done

install: proto ## install the binaries in GOPATH/bin
	@cat .asciiart.txt
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@for app in vega vegabench ; do \
		env CGO_ENABLED=0 go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" "./cmd/$$app" || exit 1 ; \
	done

gqlgen: deps ## run gqlgen
	@cd ./internal/gateway/graphql/ && go run github.com/99designs/gqlgen -c gqlgen.yml

proto: | deps ${PROTOFILES} ${PROTOVALFILES} ## build proto definitions

.PRECIOUS: proto/%.validator.pb.go
proto/%.validator.pb.go: proto/%.proto
	@protoc -Ivendor -Ivendor/github.com/google/protobuf/src -I. --govalidators_out=paths=source_relative:. "$<" && \
	sed -i -re 's/this\.Size_/this.Size/' "$@" && \
	./script/fix_imports.sh "$@"

.PRECIOUS: proto/%.pb.go
proto/%.pb.go: proto/%.proto
	@protoc -Ivendor -Ivendor/github.com/google/protobuf/src -I. --go_out=paths=source_relative:. "$<"

proto_check: deps ## proto: Check committed files match just-generated files
	@touch proto/*.proto ; \
	make proto 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only proto/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff proto/ ; \
		exit 1 ; \
	fi

# GRPC Targets

grpc: proto/api/trading.pb.go proto/api/trading.validator.pb.go proto/api/trading.pb.gw.go proto/api/trading.swagger.json  ## Generate gRPC files: grpc.pb.go, grpc.validator.pb.go, grpc.pb.gw.go, grpc.swagger.json

proto/api/trading.pb.go: proto/api/trading.proto
	@protoc -I. -Iproto -Ivendor -Ivendor/github.com/google/protobuf/src --go_out=plugins=grpc,paths=source_relative:. "$<"

proto/api/trading.validator.pb.go: proto/api/trading.proto
	@protoc -Ivendor -Ivendor/github.com/google/protobuf/src -I. \
		 --govalidators_out=paths=source_relative:. "$<" && \
	./script/fix_imports.sh "$@"

GRPC_CONF_OPT := logtostderr=true,grpc_api_configuration=internal/gateway/rest/grpc-rest-bindings.yml,paths=source_relative:.
SWAGGER_CONF_OPT := logtostderr=true,grpc_api_configuration=internal/gateway/rest/grpc-rest-bindings.yml:.

# This creates a reverse proxy to forward HTTP requests into gRPC requests
proto/api/trading.pb.gw.go: proto/api/trading.proto internal/gateway/rest/grpc-rest-bindings.yml
	@protoc -Ivendor -I. -Iproto/api/ -Ivendor/github.com/google/protobuf/src --grpc-gateway_out=$(GRPC_CONF_OPT) "$<"

# Generate Swagger documentation
proto/api/trading.swagger.json: proto/api/trading.proto internal/gateway/rest/grpc-rest-bindings.yml
	@protoc -Ivendor -Ivendor/github.com/google/protobuf/src -I. -Iinternal/api/ --swagger_out=$(SWAGGER_CONF_OPT) "$<"

grpc_check: deps ## gRPC: Check committed files match just-generated files
	@touch internal/api/*.proto ; \
	make grpc 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only internal/api/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff internal/api/ ; \
		exit 1 ; \
	fi

# Misc Targets

docker: ## Make docker container image from scratch
	@test -f "$(HOME)/.ssh/id_rsa" || exit 1
	@docker build \
		--build-arg SSH_KEY="$$(cat ~/.ssh/id_rsa)" \
		-t "registry.gitlab.com/vega-protocol/trading-core:latest" \
		.

docker_quick: build ## Make docker container image using pre-existing binaries
	@for app in vega vegabench ; do \
		f="cmd/$$app/$$app" ; \
		if ! test -f "$$f" ; then \
			echo "Failed to find: $$f" ; \
			exit 1 ; \
		fi ; \
		cp -a "$$f" . || exit 1 ; \
	done
	@docker build \
		-t "registry.gitlab.com/vega-protocol/trading-core:latest" \
		-f Dockerfile.quick \
		.
	@for app in vega vegabench ; do \
		rm -rf "./$$app" ; \
	done

gettools:
	@./script/gettools.sh

clean: ## Remove previous build
	@rm -f ./vega{,bench} ./cmd/{vega/vega,vegabench/vegabench}

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
