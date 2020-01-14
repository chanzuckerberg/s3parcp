export GO111MODULE=on

setup: ## setup development dependencies
	scripts/download-golangci-lint.sh
.PHONY: setup

lint: ## run linter
	./bin/golangci-lint run
.PHONY: lint

build:
	go build
.PHONY: build

deps: ## install dependencies
	go mod tidy
.PHONY: deps

test: deps ## run tests
	go test ./...
.PHONY: test

test-coverage: ## run tests and generate coverage report
	go test -coverprofile=coverage.out -covermode=atomic ./...
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

