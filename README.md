<hr/>
<p align="center">
  <sub><img src="https://1password.com/img/logo-v1.svg" alt="1Password" width="20" /></sub> <b>SecretHub has joined 1Password!</b> Find out more on the <a href="https://secrethub.io/blog/secrethub-joins-1password/">SecretHub blog</a>. 🎉
</p>
<hr/>

<p align="center">
  <img src="https://secrethub.io/img/github-banner.png?v4" alt="SecretHub" width="400">
</p>
<br/>

<p align="center">
  <a href="https://signup.secrethub.io/"><img alt="Get Started" src="https://secrethub.io/img/buttons/github/get-started.png?v1" height="28" /></a>
  <a href="https://secrethub.io/docs/reference/cli/"><img alt="View Docs" src="https://secrethub.io/img/buttons/github/view-docs.png?v2" height="28" /></a>
</p>
<br/>

# SecretHub CLI

[![GoDoc](https://godoc.org/github.com/secrethub/secrethub-cli?status.svg)][godoc]
[![CircleCI](https://circleci.com/gh/secrethub/secrethub-cli.svg?style=shield)][circle-ci]
[![Go Report Card](https://goreportcard.com/badge/github.com/secrethub/secrethub-cli)][goreportcard]
[![Version](https://img.shields.io/github/release/secrethub/secrethub-cli.svg)][latest-version]
[![Discord](https://img.shields.io/badge/chat-on%20discord-7289da.svg?logo=discord)][discord]

The SecretHub CLI provides the command-line interface to interact with the SecretHub API.

> [SecretHub][secrethub] is a secrets management tool that works for every engineer. Securely provision passwords and keys throughout your entire stack with just a few lines of code.

## Usage

Below you can find a selection of some of the most-used SecretHub commands. Run `secrethub --help` or the [CLI reference docs][cli-reference-docs] for a complete list of all commands.

### Reading and writing secrets
```sh
$ secrethub read <path/to/secret>
Print a secret to stdout.

$ secrethub generate <path/to/secret>
Generate a random value and store it as a new version of a secret

$ secrethub write <path/to/secret>
Ask for a value to store as a secret.

$ echo "mysecret" | secrethub write <path/to/secret>
Store a piped value as a secret.

$ secrethub write -i <filename> <path/to/secret>
Store the contents of a file as a secret.
```

### Provisioning your applications with secrets
```sh
$ export MYSECRET=secrethub://path/to/secret
$ secrethub run -- <executable/script>
Automatically load secrets into environment variables and provide them to the wrapped executable or script.

$ echo "mysecret: {{path/to/secret}}" | secrethub inject
Read a configuration template from stdin and automatically inject secrets into it.
```

### Access control
```sh
$ secrethub service init <namespace>/<repo> --permission <dir>:<read/write/admin>
Create a service account for the given repository and automatically grant read, write or admin permission on the given directory.

$ secrethub acl set <path/to/directory> <account-name> <read/write/admin>
Grant an account read, write or admin permission on a directory.

$ secrethub repo revoke <namespace>/<repo> <account-name>
Revoke an account's access to a repository.
```

## Integrations

SecretHub integrates with all the tools you already know and love.

<p align="left">
  <img src="https://secrethub.io/img/features/integrations.png" width="450px" />
</p>

Check out the [Integrations](integrations) page to find out how SecretHub works with your tools.

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



[secrethub]: https://secrethub.io/
[getting-started]: https://secrethub.io/docs/getting-started/
[cli-reference-docs]: https://secrethub.io/docs/reference/cli/
[integrations]: https://secrethub.io/integrations/
[releases]: https://github.com/secrethub/secrethub-cli/releases
[latest-version]: https://github.com/secrethub/secrethub-cli/releases/latest
[godoc]: http://godoc.org/github.com/secrethub/secrethub-cli
[goreportcard]: https://goreportcard.com/report/github.com/secrethub/secrethub-cli
[circle-ci]: https://circleci.com/gh/secrethub/secrethub-cli
[discord]: https://discord.gg/gyQXAFU
