package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// ConfigCommand handles operations on the SecretHub configuration.
type ConfigCommand struct {
	io              ui.IO
	credentialStore CredentialConfig
}

// NewConfigCommand creates a new ConfigCommand.
func NewConfigCommand(io ui.IO, store CredentialConfig) *ConfigCommand {
	return &ConfigCommand{
		io:              io,
		credentialStore: store,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ConfigCommand) Register(r command.Registerer) {
	clause := r.Command("config", "Manage your local configuration.")
	NewConfigUpdatePassphraseCommand(cmd.io, cmd.credentialStore).Register(clause)
	NewConfigUpgradeCommand().Register(clause)
}
