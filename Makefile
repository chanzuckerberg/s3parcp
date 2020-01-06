SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=false
# TODO add release flag
GO_PACKAGE=$(shell go list)
LDFLAGS=-ldflags "-w -s -X $(GO_PACKAGE)/util.GitSha=${SHA} -X $(GO_PACKAGE)/util.Version=${VERSION} -X $(GO_PACKAGE)/util.Dirty=${DIRTY}"
export GOFLAGS=-mod=vendor
export GO111MODULE=on

all: test install

setup: ## setup development dependencies
	curl -sfL https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | sh
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh
.PHONY: setup

lint: ## run the fast go linters
	./bin/reviewdog -conf .reviewdog.yml  -diff "git diff master"
.PHONY: lint

lint-ci: ## run the fast go linters
	./bin/reviewdog -conf .reviewdog.yml  -reporter=github-pr-review
.PHONY: lint-ci

lint-all: ## run the fast go linters
	# doesn't seem to be a way to get reviewdog to not filter by diff
	./bin/golangci-lint run
.PHONY: lint-all

release: ## run a release
	./bin/bff bump
	git push
	goreleaser release
.PHONY: release

release-prerelease: build ## release to github as a 'pre-release'
	version=`./s3parcp version`; \
	git tag v"$$version"; \
	git push
	git push --tags
	./bin/goreleaser release -f .goreleaser.prerelease.yml --debug
.PHONY: release-prelease

release-snapshot: ## run a release
	./bin/goreleaser release --snapshot
.PHONY: release-snapshot

build: ## build the binary
	go build ${LDFLAGS} .
.PHONY: build

deps:
	go mod tidy
	go mod vendor
.PHONY: deps

coverage: ## run the go coverage tool, reading file coverage.out
	go tool cover -html=coverage.out
.PHONY: coverage

test: deps ## run tests
	go test -cover ./...
.PHONY: test

test-ci: ## run tests
	goverage -coverprofile=coverage.out -covermode=atomic ./...
.PHONY: test-ci

test-offline:  ## run only tests that don't require internet
	go test -tags=offline ./...
.PHONY: test-offline

test-coverage:  ## run the test with proper coverage reporting
	goverage -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out
.PHONY: test-coverage

install: ## install the fogg binary in $GOPATH/bin
	go install ${LDFLAGS} .
.PHONY: install

help: ## display help for this makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

clean: ## clean the repo
	rm fogg 2>/dev/null || true
	go clean
	go clean -testcache
	rm -rf dist 2>/dev/null || true
	rm coverage.out 2>/dev/null || true

update-golden-files: clean ## update the golden files in testdata
	go test -v -run TestIntegration ./apply/ -update
.PHONY: update-golden-files
