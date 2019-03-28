package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// AccountCommand handles operations on SecretHub accounts.
type AccountCommand struct {
	io              ui.IO
	newClient       newClientFunc
	credentialStore CredentialStore
}

// NewAccountCommand creates a new AccountCommand.
func NewAccountCommand(io ui.IO, newClient newClientFunc, credentialStore CredentialStore) *AccountCommand {
	return &AccountCommand{
		io:              io,
		newClient:       newClient,
		credentialStore: credentialStore,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *AccountCommand) Register(r Registerer) {
	clause := r.Command("account", "Manage your SecretHub account.")
	NewAccountInspectCommand(cmd.io, cmd.newClient).Register(clause)
	NewAccountInitCommand(cmd.io, cmd.newClient, cmd.credentialStore).Register(clause)
	NewAccountEmailVerifyCommand(cmd.io, cmd.newClient).Register(clause)
}
