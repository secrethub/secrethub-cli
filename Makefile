commit: format lint test

format:
	@goimports -w $(find . -type f -name '*.go')

lint:
	@golangci-lint run

test:
	@go test ./...

tools: format-tools lint-tools

format-tools:
	@go get -u golang.org/x/tools/cmd/goimports

lint-tools:
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.15.0

COMMIT=`git rev-parse --short HEAD`
VERSION=`git describe --always`
BUILD_FLAGS=-ldflags "-s -w -X main.commit=${COMMIT} -X main.version=${VERSION}" -tags=production

build:
	go build ${BUILD_FLAGS} ./cmd/secrethub

install:
	go install ${BUILD_FLAGS} ./cmd/secrethub

msi:
	./msi.sh
