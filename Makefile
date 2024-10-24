# Makefile

.PHONY: all
all: build

.PHONY: lint
lint: ## Lint the files
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 run -v --config .golangci.toml

.PHONY: retest
retest: ## Re-run all unit tests
	go test -v -count=1 ./...

.PHONY: test
test: ## Run unit tests
	go test -v -failfast ./...

.PHONY: integrationtest
integrationtest: ## run integration tests, showing ledger movements and full scenario output
	go test -v ./core/integration/... --godog.format=pretty --godog.tags=VAMM3

.PHONY: gqlgen
gqlgen:
	cd datanode/gateway/graphql && go run github.com/99designs/gqlgen@v0.17.45 --config=gqlgen.yml

.PHONY: race
race: ## Run data race detector
	go test -v -race ./...

.PHONY: mocks
mocks: ## Make mocks
	go generate -v ./...

.PHONY: mocks_check
mocks_check: ## mocks: Check committed files match just-generated files
# TODO: how to delete all generated files
#	@make proto_clean 1>/dev/null
	@make mocks 1>/dev/null
	@files="$$(git diff --name-only)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files: " $$files ; \
		git diff ; \
		exit 1 ; \
	fi

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
		pipelinecomponents/markdown-spellcheck \
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
	buf generate

.PHONY: proto_docs
proto_docs: ## build proto definitions
	rm -rf protos/generated
	buf generate --template buf.gen.swagger.yaml

.PHONY: proto_check
proto_check: ## proto: Check committed files match just-generated files
	@make proto_clean 1>/dev/null
	@make proto 1>/dev/null
	@files="$$(git diff --name-only protos/vega/ protos/data-node/)" ; \
	if test -n "$$files" ; then \
		echo "Committed files do not match just-generated files: " $$files ; \
		git diff ; \
		exit 1 ; \
	fi

.PHONY: proto_format_check
proto_format_check:
	@make proto_clean 1>/dev/null
	@make proto 1>/dev/null
	buf format --exit-code --diff

.PHONY: proto_clean
proto_clean:
	@find protos/vega protos/data-node -name '*.pb.go' -o -name '*.pb.gw.go' \
		| xargs -r rm

.PHONY: buflint
buflint: ## Run buf lint
	@buf lint
