# SecretHub CLI

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)][godoc]
[![Travis CI](https://travis-ci.org/secrethub/secrethub-cli.svg?branch=master)][travis-ci]
[![GolangCI](https://golangci.com/badges/github.com/secrethub/secrethub-cli.svg)][golang-ci]
[![Go Report Card](https://goreportcard.com/badge/github.com/secrethub/secrethub-cli)][goreportcard]

<img src="https://secrethub.io/img/secrethub-logo-shield.svg" alt="SecretHub" width="100px"/>

---

[SecretHub][secrethub] is a developer tool to help you keep database passwords, API tokens, and other secrets out of IT automation scripts.

`secrethub-cli` provides the command line interface to interact with the SecretHub API.

## Installation

Binaries are provided in the [GitHub releases][releases]. Place the binary of your OS in the PATH and you are good to go!

We plan to distribute secrethub via package managers ([#27](https://github.com/secrethub/secrethub-cli/issues/27)).
Please feel free to join the discussion and let us know what package manager you are using.

## Usage

Checkout the [getting started docs](https://secrethub.io/docs/getting-started/).
Or have a look at the [reference docs](https://secrethub.io/docs/reference/) when you are looking for documentation about a specific command.

## Development

Pull requests from the community are welcome.
If you'd like to contribute, please checkout [the contributing guidelines](./CONTRIBUTING.md).

## Test

Run all tests:

    make test

Run tests for one package:

    go test ./internals/secrethub

Run a single test:

    go test ./internals/secrethub -run TestWriteCommand_Run

[secrethub]: https://secrethub.io
[releases]: https://github.com/secrethub/secrethub-cli/releases
[godoc]: http://godoc.org/github.com/secrethub/secrethub-cli
[golang-ci]: https://golangci.com/r/github.com/secrethub/secrethub-cli
[goreportcard]: https://goreportcard.com/report/github.com/secrethub/secrethub-cli
[travis-ci]: https://travis-ci.org/secrethub/secrethub-cli
