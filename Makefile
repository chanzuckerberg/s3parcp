export GO111MODULE=on

all: test install

setup: ## setup development dependencies
	curl -sfL https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | sh
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh
.PHONY: setup

lint:
	./bin/golangci-lint run
.PHONY: lint

release: ## run a release
	./bin/bff bump
	git push
	goreleaser release
.PHONY: release

release-prerelease:
	./bin/bff bump
	git push
	./bin/goreleaser release -f .goreleaser.prerelease.yml --debug
.PHONY: release-prelease

release-snapshot: ## run a release
	./bin/goreleaser release --snapshot
.PHONY: release-snapshot

build:
	go build
.PHONY: build

deps:
	go mod tidy
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

install: ## install the s3parcp binary in $GOPATH/bin
	go install ${LDFLAGS} .
.PHONY: install

help: ## display help for this makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

clean: ## clean the repo
	rm s3parcp 2>/dev/null || true
	go clean
	go clean -testcache
	rm -rf dist 2>/dev/null || true
	rm coverage.out 2>/dev/null || true

