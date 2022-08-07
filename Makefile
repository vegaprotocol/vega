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
	go test -v -failfast ./...

.PHONY: integrationtest
integrationtest: ## run integration tests, showing ledger movements and full scenario output
	go test -v ./core/integration/... --godog.format=pretty

.PHONY: gqlgen
gqlgen:
	cd datanode/gateway/graphql && go run github.com/99designs/gqlgen --config=gqlgen.yml

.PHONY: race
race: ## Run data race detector
	go test -v -race ./...

.PHONY: mocks
mocks: ## Make mocks
	go generate ./...

.PHONY: build
build: ## install the binaries in cmd/{progname}/
	go build -o cmd/vega ./cmd/vega


.PHONY: gen-contracts-code
gen-contracts-code:
	cd contracts && ./gen.sh
.PHONY: install
install: ## install the binaries in GOPATH/bin
	go install ./...

codeowners_check:
	@if grep -v '^#' CODEOWNERS | grep "," ; then \
		echo "CODEOWNERS cannot have entries with commas" ; \
		exit 1 ; \
	fi

# Make sure the mdspell command matches the one in .drone.yml.
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


.PHONY: clean
clean: SHELL:=/bin/bash
clean: ## Remove previous build
	rm cmd/vega/vega
	rm -rf ./**/*-re

.PHONY: proto
proto: ## build proto definitions
	@./script/generate.sh

.PHONY: proto_json
proto_json: ## build proto definitions
	@./script/generate_json.sh

.PHONY: proto_check
proto_check: ## proto: Check committed files match just-generated files
	@make proto_clean 1>/dev/null
	@make proto 1>/dev/null
	@files="$$(git diff --name-only protos/vega/ protos/data-node/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files: " $$files ; \
		test -n "$(CI)" && git diff vega/ ; \
		exit 1 ; \
	fi

.PHONY: proto_clean
proto_clean:
	@find protos/vega protos/data-node -name '*.pb.go' -o -name '*.pb.gw.go' \
		| xargs -r rm

.PHONY: buflint
buflint: ## Run buf lint
	@buf lint
