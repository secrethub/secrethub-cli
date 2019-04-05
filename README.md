<p align="center">
  <img src="https://secrethub.io/img/secrethub-logo.svg" alt="SecretHub" width="380px"/>
</p>
<h1 align="center">
  <i>CLI</i>
</h1>

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)][godoc]
[![Travis CI](https://travis-ci.org/secrethub/secrethub-cli.svg?branch=master)][travis-ci]
[![GolangCI](https://golangci.com/badges/github.com/secrethub/secrethub-cli.svg)][golang-ci]
[![Go Report Card](https://goreportcard.com/badge/github.com/secrethub/secrethub-cli)][goreportcard]
[![Version]( https://img.shields.io/github/release/secrethub/secrethub-cli.svg)][latest-version]
[![Discord](https://img.shields.io/badge/chat-on%20discord-7289da.svg?logo=discord)][discord]

The SecretHub CLI provides the command-line interface to interact with SecretHub.

> [SecretHub][secrethub] is a developer tool to help you keep database passwords, API tokens, and other secrets out of IT automation scripts.

## Usage

```sh
$ secrethub write path/to/secret
Type in your secret: ************************  

$ cat config.yml.tpl
db_user: myapp
db_password: ${path/to/secret:latest}

$ cat config.yml.tpl | secrethub inject
db_user: myapp
db_password: LEYkTdMCksCVMc4X3gpYN0fk
```

## Installation

The SecretHub CLI can be installed in various ways. Have a look at our [installation guide](https://secrethub.io/docs/getting-started/install) for more information.

## Getting started

Checkout the [getting started docs](https://secrethub.io/docs/getting-started/).
Or have a look at the [reference docs](https://secrethub.io/docs/reference/) where each command is documented in detail.

## Development

Pull requests from the community are welcome.
If you'd like to contribute, please checkout [the contributing guidelines](./CONTRIBUTING.md).

### Build

To build from source, having [Golang](https://golang.org) installed is required.
To build the binary in the current directory, run:

```sh
make build
```

### Install

To install the binary in the [GOBIN](https://golang.org/cmd/go/#hdr-GOPATH_environment_variable) directory, run:
```sh
make install
```

### Test

Run all tests:

    make test

Run tests for one package:

    go test ./internals/secrethub

Run a single test:

    go test ./internals/secrethub -run TestWriteCommand_Run

## Getting help

Come chat with us on [Discord][discord] or email us at [support@secrethub.io](mailto:support@secrethub.io)

[secrethub]: https://secrethub.io
[releases]: https://github.com/secrethub/secrethub-cli/releases
[latest-version]: https://github.com/secrethub/secrethub-cli/releases/latest
[godoc]: http://godoc.org/github.com/secrethub/secrethub-cli
[golang-ci]: https://golangci.com/r/github.com/secrethub/secrethub-cli
[goreportcard]: https://goreportcard.com/report/github.com/secrethub/secrethub-cli
[travis-ci]: https://travis-ci.org/secrethub/secrethub-cli
[discord]: https://discord.gg/gyQXAFU
