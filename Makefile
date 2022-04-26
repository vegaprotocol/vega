# Makefile

.PHONY: all
all: build

.PHONY: generate
generate: gqlgen mocks ci_check build

.PHONY: lint
lint: ## Lint the files
	golangci-lint run --config .golangci.toml

.PHONY: retest
retest: ## Re-run all uni tests
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

.PHONY: coverage
coverage: ## Generate global code coverage report
	@./script/build.sh -a coverage

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

.PHONY: gqlgen
gqlgen: ## run gqlgen
	@./script/build.sh -a gqlgen

.PHONY: gqlgen_check
gqlgen_check: ## GraphQL: Check committed files match just-generated files
	@find gateway/graphql -name '*.graphql' -o -name '*.yml' -exec touch '{}' ';' ; \
	make gqlgen 1>/dev/null || exit 1 ; \
	files="$$(git diff --name-only gateway/graphql/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files:" $$files ; \
		test -n "$(CI)" && git diff gateway/graphql/ ; \
		exit 1 ; \
	fi

# Misc Targets

codeowners_check:
	@if grep -v '^#' CODEOWNERS | grep "," ; then \
		echo "CODEOWNERS cannot have entries with commas" ; \
		exit 1 ; \
	fi

.PHONY: gettools_develop
gettools_develop:
	@./script/gettools.sh develop

.PHONY: spellcheck
spellcheck: ## Run markdown spellcheck container
	@docker run --rm -ti \
		--entrypoint mdspell \
		-v "$(PWD):/src" \
		ghcr.io/vegaprotocol/devops-infra/markdownspellcheck:latest \
			--en-gb \
			--ignore-acronyms \
			--ignore-numbers \
			--no-suggestions \
			--report \
			'*.md' \
			'docs/**/*.md'

.PHONY: yamllint
yamllint:
	git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'

# Do a bunch of the checks the CI does, to help you catch them before commit
ci_check: spellcheck yamllint lint test

.PHONY: buflint
buflint: ## Run buf lint
	@./script/build.sh -a buflint

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
