commit: format lint test

format:
	@goimports -w $(find . -type f -name '*.go')

lint:
	@golangci-lint run

test:
	@go test -race ./...

tools: format-tools lint-tools

format-tools:
	@go get -u golang.org/x/tools/cmd/goimports

GOLANGCI_VERSION=v1.23.8

lint-tools:
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_VERSION}

COMMIT=`git rev-parse --short HEAD`
VERSION=`git describe --always`
BUILD_FLAGS=-ldflags "-s -w -X "github.com/secrethub/secrethub-cli/internals/secrethub.Commit=${COMMIT}" -X "github.com/secrethub/secrethub-cli/internals/secrethub.Version=${VERSION}"" -tags=production

build:
	go build ${BUILD_FLAGS} ./cmd/secrethub

install:
	go install ${BUILD_FLAGS} ./cmd/secrethub
