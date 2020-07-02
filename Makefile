commit: format lint test

format:
	@goimports -w $(find . -type f -name '*.go')

GOLANGCI_VERSION=v1.23.8
lint:
	@docker run --rm -t -v $$(go env GOCACHE):/cache/go -e GOCACHE=/cache/go -e GOLANGCI_LINT_CACHE=/cache/go -v $$(go env GOPATH)/pkg:/go/pkg -v ${PWD}:/app -w /app golangci/golangci-lint:${GOLANGCI_VERSION}-alpine golangci-lint run ./...

test:
	@go test -race ./...

tools: format-tools lint-tools

format-tools:
	@go get -u golang.org/x/tools/cmd/goimports

COMMIT=`git rev-parse --short HEAD`
VERSION=`git describe --always`
BUILD_FLAGS=-ldflags "-s -w -X "github.com/secrethub/secrethub-cli/internals/secrethub.Commit=${COMMIT}" -X "github.com/secrethub/secrethub-cli/internals/secrethub.Version=${VERSION}"" -tags=production

build:
	go build ${BUILD_FLAGS} ./cmd/secrethub

install:
	go install ${BUILD_FLAGS} ./cmd/secrethub
