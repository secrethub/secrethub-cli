package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// CredentialCommand handles operations on SecretHub credentials.
type CredentialCommand struct {
	io              ui.IO
	clientFactory   ClientFactory
	credentialStore CredentialConfig
}

// NewCredentialCommand creates a new CredentialCommand.
func NewCredentialCommand(io ui.IO, clientFactory ClientFactory, credentialStore CredentialConfig) *CredentialCommand {
	return &CredentialCommand{
		io:              io,
		clientFactory:   clientFactory,
		credentialStore: credentialStore,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *CredentialCommand) Register(r cli.Registerer) {
	clause := r.Command("credential", "Manage your credentials.")
	NewCredentialListCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
	NewCredentialBackupCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
	NewCredentialDisableCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
	NewCredentialUpdatePassphraseCommand(cmd.io, cmd.credentialStore).Register(clause)
}
