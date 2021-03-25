#! /usr/bin/make
#

SHELL=/bin/bash -euo pipefail
NAME=$(shell find cmd -name "*.go" -exec dirname {} \; | sed -e 's|cmd/||')

GOPATH=$(shell go env GOPATH)
GOBIN=$(GOPATH)/bin

default: $(NAME)
.PHONY: default

vendor: go.mod go.sum
	go mod download
	go mod vendor

lint: vendor golangci-lint-config.yaml
	@PATH=$(GOBIN):${PATH} golangci-lint run --config golangci-lint-config.yaml
.PHONY: lint

$(NAME): vendor $(shell find . -name \*.go)
	go build -o bin/$@ ./cmd/$@

build: $(NAME)
.PHONY: build

test: vendor
	go test -race -v ./...
.PHONY: test

clean:
	rm -rf bin vendor
.PHONY: clean

all: lint test build
.PHONY: all
