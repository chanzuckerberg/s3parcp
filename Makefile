export GO111MODULE=on

setup: ## setup development dependencies
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
.PHONY: setup

lint: ## run linter
	./bin/golangci-lint run
.PHONY: lint

release: ## run a release
	./bin/goreleaser release
.PHONY: release

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

test-coverage:  ## run the test with proper coverage reporting
	goverage -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out
.PHONY: test-coverage

install: ## install the s3parcp binary in $GOPATH/bin
	go install .
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

