# SecretHub CLI

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)][godoc]
[![Travis CI](https://travis-ci.org/secrethub/secrethub-cli.svg?branch=master)][travis-ci]
[![GolangCI](https://golangci.com/badges/github.com/secrethub/secrethub-cli.svg)][golang-ci]
[![Go Report Card](https://goreportcard.com/badge/github.com/secrethub/secrethub-cli)][goreportcard]
[![Discord](https://img.shields.io/badge/discord-SecretHub-7289da.svg?logo=discord)][discord-cli]

<img src="https://secrethub.io/img/secrethub-logo-shield.svg" alt="SecretHub" width="100px"/>

---

[SecretHub][secrethub] is a developer tool to help you keep database passwords, API tokens, and other secrets out of IT automation scripts.

`secrethub-cli` provides the command line interface to interact with the SecretHub API.

1. [Installation](#installation)
1. [Test your installation](#test-your-installation)
1. [Install auto-completion](#install-auto-completion)
1. [Getting Started](#getting-started)
1. [Development](#development)
1. [Uninstalling](#uninstalling)
1. [Getting Help](#getting-help)

## Installation

### Download a binary distribution

Official binary distributions are available for the Linux, macOS, and Windows operating systems for both the 32-bit (386) and 64-bit (amd64) versions.
You can find the latest release [here][releases].

To install the SecretHub CLI, download the archive file appropriate for your operating system and extract it e.g. to `/usr/local/secrethub`.

```sh
mkdir /usr/local/secrethub
tar -C /usr/local/secrethub -xzf secrethub-VERSION-OS-ARCH.tar.gz
```

Ensure it is accessible through the `PATH` environment variable.
```sh
export PATH=$PATH:/usr/local/secrethub
```

### Build from source

To build from source, [GoLang](https://golang.org) is required.

To install the binary in the [GOBIN](https://golang.org/cmd/go/#hdr-GOPATH_environment_variable), run:
```sh
make install
```

Alternatively, to build the binary in the current directory, run:
```sh
make build
```

Now you can move it into the `PATH` to use it from any directory:
```sh
mv ./secrethub /usr/local/bin/
```

### Use a package manager

We plan to distribute secrethub via package managers ([#27](https://github.com/secrethub/secrethub-cli/issues/27)).
Please feel free to join the discussion and let us know what package manager you are using.

## Test your installation

Verify your installation works by running the following command:
```sh
secrethub --version
```

## Install auto-completion

To install auto completion for the CLI, run one of the following commands depending on your shell of choice:

```sh
# Install bash completion
secrethub --completion-script-bash > /etc/bash_completion.d/secrethub
```
```sh
# Install zsh completion
secrethub --completion-script-zsh > ~/.zsh/completion/secrethub
```

## Getting started

Checkout the [getting started docs](https://secrethub.io/docs/getting-started/).
Or have a look at the [reference docs](https://secrethub.io/docs/reference/) where each command is documented in detail.

## Development

Pull requests from the community are welcome.
If you'd like to contribute, please checkout [the contributing guidelines](./CONTRIBUTING.md).

### Test

Run all tests:

    make test

Run tests for one package:

    go test ./internals/secrethub

Run a single test:

    go test ./internals/secrethub -run TestWriteCommand_Run

## Uninstalling

To remove an existing SecretHub installation from your system delete the secrethub directory.
This is usually `/usr/local/secrethub` under Linux, macOS, and FreeBSD or `c:\SecretHub` under Windows.

You should also remove the SecretHub directory from your PATH environment variable.

If you've installed auto-completion, you should remove either `/etc/bash_completion.d/secrethub` or `~/.zsh/completion/secrethub`.

## Getting help

Please reach out on [Discord][discord-cli] or via email ([support@secrethub.io](mailto:support@secrethub.io)) if you have any questions. We're here to help.

[secrethub]: https://secrethub.io
[releases]: https://github.com/secrethub/secrethub-cli/releases
[godoc]: http://godoc.org/github.com/secrethub/secrethub-cli
[golang-ci]: https://golangci.com/r/github.com/secrethub/secrethub-cli
[goreportcard]: https://goreportcard.com/report/github.com/secrethub/secrethub-cli
[travis-ci]: https://travis-ci.org/secrethub/secrethub-cli
[discord-cli]: https://discord.gg/Wut4KUK
