<p align="center">
  <img src="https://secrethub.io/img/secrethub-logo.svg" alt="SecretHub" width="380px"/>
</p>
<h1 align="center">
  <i>CLI</i>
</h1>

[![GoDoc](https://godoc.org/github.com/secrethub/secrethub-cli?status.svg)][godoc]
[![CircleCI](https://circleci.com/gh/secrethub/secrethub-cli.svg?style=shield)][circle-ci]
[![Go Report Card](https://goreportcard.com/badge/github.com/secrethub/secrethub-cli)][goreportcard]
[![Version]( https://img.shields.io/github/release/secrethub/secrethub-cli.svg)][latest-version]
[![Discord](https://img.shields.io/badge/chat-on%20discord-7289da.svg?logo=discord)][discord]

The SecretHub CLI provides the command-line interface to interact with the SecretHub API.

> [SecretHub][secrethub] is an end-to-end encrypted secret management service that helps developers keep database passwords, API keys, and other secrets out of source code.

## Usage

```sh
$ secrethub write path/to/secret
Type in your secret: ************************  

$ cat config.yml.tpl
db_user: myapp
db_password: {{ path/to/secret:latest }}

$ cat config.yml.tpl | secrethub inject
db_user: myapp
db_password: LEYkTdMCksCVMc4X3gpYN0fk
```

See the [reference docs][reference-docs] for a detailed overview of all commands.

## Get started

### 1. [Download][installation-guide] the CLI.  

Official distributions are available for Linux, macOS, and Windows for both `386` (32-bit) and `amd64` (64-bit) architectures.

Check out the [installation guide][installation-guide] for detailed instructions on how to install the SecretHub CLI on your platform of choice.

### 2. Run `signup`

Run `signup` to claim your free developer account:

```
secrethub signup
```

And you're done. 
Follow the [getting started guide][getting-started] for a brief introduction into the basics of SecretHub.

## Getting help

Come chat with us on [Discord][discord] or email us at [support@secrethub.io](mailto:support@secrethub.io)

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



[secrethub]: https://secrethub.io
[getting-started]: https://secrethub.io/docs/getting-started/
[installation-guide]: https://secrethub.io/docs/getting-started/install
[reference-docs]: https://secrethub.io/docs/reference/
[releases]: https://github.com/secrethub/secrethub-cli/releases
[latest-version]: https://github.com/secrethub/secrethub-cli/releases/latest
[godoc]: http://godoc.org/github.com/secrethub/secrethub-cli
[goreportcard]: https://goreportcard.com/report/github.com/secrethub/secrethub-cli
[circle-ci]: https://circleci.com/gh/secrethub/secrethub-cli
[discord]: https://discord.gg/gyQXAFU
