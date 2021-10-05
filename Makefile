# Makefile

.PHONY: all
all: build

.PHONY: lint
lint: ## Lint the files
	@t="$$(mktemp)" ; \
	go list ./... | xargs golint | grep -vE '(and that stutters|blank import should be|should have comment|which can be annoying to use)' | tee "$$t" ; \
	code=0 ; test "$$(wc -l <"$$t" | awk '{print $$1}')" -gt 0 && code=1 ; \
	rm -f "$$t" ; \
	exit "$$code"

.PHONY: retest
retest: ## Re-run all unit tests
	@./script/build.sh -a retest

.PHONY: test
test: ## Run unit tests
	@./script/build.sh -a test

.PHONY: integrationtest
integrationtest: ## run integration tests, showing ledger movements and full scenario output
	@./script/build.sh -a integrationtest

.PHONY: spec_feature_test
spec_feature_test: ## run integration tests in the specs internal repo
	@specsrepo="$(PWD)/../specs-internal" \
	./script/build.sh -a spec_feature_test

.PHONY: race
race: ## Run data race detector
	@./script/build.sh -a race

.PHONY: mocks
mocks: ## Make mocks
	@./script/build.sh -a mocks

.PHONY: msan
msan: ## Run memory sanitizer
	@if ! which clang 1>/dev/null ; then echo "Need clang" ; exit 1 ; fi
	@env CC=clang CGO_ENABLED=1 go test -msan ./...

.PHONY: vet
vet: ## Run go vet
	@./script/build.sh -a vet

.PHONY: coverage
coverage: ## Generate global code coverage report
	@./script/build.sh -a coverage

.PHONY: deps
deps: ## Get the dependencies
	@./script/build.sh -a deps

.PHONY: build
build: ## install the binaries in cmd/{progname}/
	@d="" ; test -n "$$DEBUGVEGA" && d="-d" ; \
	./script/build.sh $$d -a build -t default

.PHONY: gofmtsimplify
gofmtsimplify:
	@find . -path vendor -prune -o \( -name '*.go' -and -not -name '*_test.go' -and -not -name '*_mock.go' \) -print0 | xargs -0r gofmt -s -w

.PHONY: install
install: ## install the binaries in GOPATH/bin
	@./script/build.sh -a install -t default


.PHONY: ineffectassign
ineffectassign: ## Check for ineffectual assignments
	@ia="$$(env GO111MODULE=auto ineffassign . | grep -v '_test\.go:')" ; \
	if test "$$(echo -n "$$ia" | wc -l | awk '{print $$1}')" -gt 0 ; then echo "$$ia" ; exit 1 ; fi

codeowners_check:
	@if grep -v '^#' CODEOWNERS | grep "," ; then \
		echo "CODEOWNERS cannot have entries with commas" ; \
		exit 1 ; \
	fi

.PHONY: print_check
print_check: ## Check for fmt.Print functions in Go code
	@f="$$(mktemp)" && \
	find -name vendor -prune -o \
		-name cmd -prune -o \
		-name 'json.go' -prune -o \
		-name 'print.go' -prune -o \
		-name '*_test.go' -prune -o \
		-name 'flags.go' -prune -o \
		-name '*.go' -print0 | \
		xargs -0 grep -E '^([^/]|/[^/])*fmt.Print' | \
		tee "$$f" && \
	count="$$(wc -l <"$$f")" && \
	rm -f "$$f" && \
	if test "$$count" -gt 0 ; then exit 1 ; fi

.PHONY: gettools_develop
gettools_develop:
	@./script/gettools.sh develop

# Make sure the mdspell command matches the one in .drone.yml.
.PHONY: spellcheck
spellcheck: ## Run markdown spellcheck container
	@docker run --rm -ti \
		--entrypoint mdspell \
		-v "$(PWD):/src" \
		docker.pkg.github.com/vegaprotocol/devops-infra/markdownspellcheck:latest \
			--en-gb \
			--ignore-acronyms \
			--ignore-numbers \
			--no-suggestions \
			--report \
			'*.md' \
			'docs/**/*.md'

# The integration directory is special, and contains a package called core_test.
.PHONY: staticcheck
staticcheck: ## Run statick analysis checks
	@./script/build.sh -a staticcheck

.PHONY: misspell
misspell: # Run go specific misspell checks
	@./script/build.sh -a misspell

.PHONY: semgrep
semgrep: ## Run semgrep static analysis checks
	@./script/build.sh -a semgrep

.PHONY: clean
clean: SHELL:=/bin/bash
clean: ## Remove previous build
	@source ./script/build.sh && \
	rm -f cmd/*/*.log && \
	for app in "$${apps[@]}" ; do \
		rm -f "$$app" "cmd/$$app/$$app" "cmd/$$app/$$app"-* ; \
	done

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
