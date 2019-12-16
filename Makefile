APPS := dummyriskmodel vega vegaccount vegastream

ifeq ($(CI),)
	# Not in CI
	VERSION := dev-$(USER)
	VERSION_HASH := $(shell git rev-parse HEAD | cut -b1-8)
else
	# In CI
	ifneq ($(GITLAB_CI),)
		# In Gitlab: https://docs.gitlab.com/ce/ci/variables/predefined_variables.html

		ifneq ($(CI_COMMIT_TAG),)
			VERSION := $(CI_COMMIT_TAG)
		else
			# No tag, so make one
			VERSION := $(shell git describe --tags 2>/dev/null)
		endif
		VERSION_HASH := $(CI_COMMIT_SHORT_SHA)

	else ifneq ($(DRONE),)
		# In Drone: https://docker-runner.docs.drone.io/configuration/environment/variables/

		ifneq ($(DRONE_TAG),)
			VERSION := $(DRONE_TAG)
		else
			# No tag, so make one
			VERSION := $(shell git describe --tags 2>/dev/null)
		endif
		VERSION_HASH := $(shell echo "$(CI_COMMIT_SHA)" | cut -b1-8)

	else
		# In an unknown CI
		VERSION := unknown-CI
		VERSION_HASH := unknown-CI
	endif
endif

.PHONY: all bench deps build clean docker docker_quick grpc grpc_check help test lint mocks

all: build

lint: ## Lint the files
	@t="$$(mktemp)" ; \
	go list ./... | xargs golint | grep -vE '(and that stutters|blank import should be|should have comment|which can be annoying to use)' | tee "$$t" ; \
	code=0 ; test "$$(wc -l <"$$t" | awk '{print $$1}')" -gt 0 && code=1 ; \
	rm -f "$$t" ; \
	exit "$$code"

test: ## Run unit tests
	@go test ./...

integrationtest: ## run integration tests, showing ledger movements and full scenario output
	@go test -v ./integration/... -godog.format=pretty

race: ## Run data race detector
	@env CGO_ENABLED=1 go test -race ./...

mocks: ## Make mocks
	@go generate ./...

msan: ## Run memory sanitizer
	@if ! which clang 1>/dev/null ; then echo "Need clang" ; exit 1 ; fi
	@env CC=clang CGO_ENABLED=1 go test -msan ./...

vet: ## Run go vet
	@go vet -all ./...

vetshadow: # Run go vet with shadow detection
	@go vet -shadow ./... 2>&1 | grep -vE '^(#|gateway/graphql/generated.go|proto/.*\.pb\.(gw\.)?go)' ; \
	code="$$?" ; test "$$code" -ne 0

.PHONY: .testCoverage.txt
.testCoverage.txt:
	@go list ./... |grep -v '/gateway' | xargs go test -covermode=count -coverprofile="$@"
	@go tool cover -func="$@"

coverage: .testCoverage.txt ## Generate global code coverage report

.PHONY: .testCoverage.html
.testCoverage.html: .testCoverage.txt
	@go tool cover -html="$^" -o "$@"

coveragehtml: .testCoverage.html ## Generate global code coverage report in HTML

deps: ## Get the dependencies
	@go mod download
	@go mod vendor
	@grep 'google/protobuf' go.mod | awk '{print "# " $$1 " " $$2 "\n"$$1"/src";}' >> vendor/modules.txt
	@modvendor -copy="**/*.proto"

build: ## install the binaries in cmd/{progname}/
	@v="${VERSION}" vh="${VERSION_HASH}" gcflags="" suffix="" ; \
	if test -n "$$DEBUGVEGA" ; then \
		gcflags="all=-N -l" ; \
		suffix="-dbg" ; \
		v="debug-$$v" ; \
	fi ; \
	ldflags="-X main.Version=$$v -X main.VersionHash=$$vh" ; \
	echo "Version: $$v ($$vh)" ; \
	for app in $(APPS) ; do \
		env CGO_ENABLED=0 go build -v \
			-ldflags "$$ldflags" \
			-gcflags "$$gcflags" \
			-o "./cmd/$$app/$$app$$suffix" "./cmd/$$app" \
			|| exit 1 ; \
	done

.PHONY: gofmtsimplify
gofmtsimplify:
	@find . -path vendor -prune -o \( -name '*.go' -and -not -name '*_test.go' -and -not -name '*_mock.go' \) -print0 | xargs -0r gofmt -s -w

install: ## install the binaries in GOPATH/bin
	@cat .asciiart.txt
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@for app in $(APPS) ; do \
		env CGO_ENABLED=0 go install -v -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" "./cmd/$$app" || exit 1 ; \
	done

gqlgen: ## run gqlgen
	@cd ./gateway/graphql/ && go run github.com/99designs/gqlgen -c gqlgen.yml

gqlgen_check: ## GraphQL: Check committed files match just-generated files
	@find gateway/graphql -name '*.graphql' -o -name '*.yml' -exec touch '{}' ';' ; \
	make gqlgen 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only gateway/graphql/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff gateway/graphql/ ; \
		exit 1 ; \
	fi

ineffectassign: ## Check for ineffectual assignments
	@ia="$$(env GO111MODULE=auto ineffassign . | grep -v '_test\.go:')" ; \
	if test "$$(echo -n "$$ia" | wc -l | awk '{print $$1}')" -gt 0 ; then echo "$$ia" ; exit 1 ; fi

.PHONY: proto
proto: deps ## build proto definitions
	@./proto/generate.sh

.PHONY: proto_check
proto_check: ## proto: Check committed files match just-generated files
	@make proto 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only proto/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff proto/ ; \
		exit 1 ; \
	fi

.PHONY: proto_clean
proto_clean:
	@find proto -name '*.pb.go' -o -name '*.pb.gw.go' -o -name '*.validator.pb.go' -o -name '*.swagger.json' \
		| xargs -r rm
	@find proto/doc -name index.html -o -name index.md \
		| xargs -r rm

.PHONY: rest_check
rest_check: gateway/rest/grpc-rest-bindings.yml proto/api/trading.swagger.json
	@python3 script/check_rest_endpoints.py \
		--bindings gateway/rest/grpc-rest-bindings.yml \
		--swagger proto/api/trading.swagger.json

# Misc Targets

print_check: ## Check for fmt.Print functions in Go code
	@f="$$(mktemp)" && \
	find -name vendor -prune -o \
		-name cmd -prune -o \
		-name '*_test.go' -prune -o \
		-name '*.go' -print0 | \
		xargs -0 grep -E '^([^/]|/[^/])*fmt.Print' | \
		tee "$$f" && \
	count="$$(wc -l <"$$f")" && \
	rm -f "$$f" && \
	if test "$$count" -gt 0 ; then exit 1 ; fi

docker: ## Make docker container image from scratch
	@test -f "$(HOME)/.ssh/id_rsa" || exit 1
	@docker build \
		--build-arg SSH_KEY="$$(cat ~/.ssh/id_rsa)" \
		-t "registry.gitlab.com/vega-protocol/trading-core:$(VERSION)" \
		.

docker_quick: build ## Make docker container image using pre-existing binaries
	@for app in $(APPS) ; do \
		f="cmd/$$app/$$app" ; \
		if ! test -f "$$f" ; then \
			echo "Failed to find: $$f" ; \
			exit 1 ; \
		fi ; \
		cp -a "$$f" . || exit 1 ; \
	done
	@docker build \
		-t "registry.gitlab.com/vega-protocol/trading-core:$(VERSION)" \
		-f Dockerfile.quick \
		.
	@for app in $(APPS) ; do \
		rm -rf "./$$app" ; \
	done

.PHONY: gettools_build
gettools_build:
	@./script/gettools.sh build

.PHONY: gettools_develop
gettools_develop:
	@./script/gettools.sh develop

# Make sure the mdspell command matches the one in .drone.yml.
spellcheck: ## Run markdown spellcheck container
	@docker run --rm -ti \
		--entrypoint mdspell \
		-v "$(PWD):/src" \
		registry.gitlab.com/vega-protocol/devops-infra/markdownspellcheck:latest \
			--en-gb \
			--ignore-acronyms \
			--ignore-numbers \
			--no-suggestions \
			--report \
			'*.md' \
			'design/**/*.md'

# The integration directory is special, and contains a package called core_test.
staticcheck: ## Run statick analysis checks
	@go list ./... | grep -v /integration | xargs staticcheck
	@f="$$(mktemp)" && find integration -name '*.go' | xargs staticcheck | grep -v 'could not load export data' | tee "$$f" && \
	count="$$(wc -l <"$$f")" && rm -f "$$f" && if test "$$count" -gt 0 ; then exit 1 ; fi

clean: ## Remove previous build
	@for app in $(APPS) ; do rm -f "$$app" "cmd/$$app/$$app" "cmd/$$app/$$app-dbg" ; done

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
