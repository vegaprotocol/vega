# Makefile

.PHONY: all
all: build

.PHONY: lint
lint: ## Lint the files
	golangci-lint run -v --config .golangci.toml

.PHONY: retest
retest: ## Re-run all unit tests
	go test -v -count=1 ./...

.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: integrationtest
integrationtest: ## run integration tests, showing ledger movements and full scenario output
	go test -v ./integration/... --godog.format=pretty

.PHONY: race
race: ## Run data race detector
	go test -v -race ./...

.PHONY: mocks
mocks: ## Make mocks
	go generate ./...

.PHONY: build
build: ## install the binaries in cmd/{progname}/
	go build -o cmd/vega ./cmd/vega 
	go build -o cmd/vega ./cmd/vegabenchmark

.PHONY: install
install: ## install the binaries in GOPATH/bin
	go install ./...

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


.PHONY: clean
clean: SHELL:=/bin/bash
clean: ## Remove previous build
	rm cmd/vega/vega
	rm cmd/vega/vegabenchmark
