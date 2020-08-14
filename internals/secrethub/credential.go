package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
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
func (cmd *CredentialCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("credential", "Manage your credentials.")
	NewCredentialListCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
	NewCredentialBackupCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
	NewCredentialDisableCommand(cmd.io, cmd.clientFactory.NewClient).Register(clause)
}
