package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// ImportCommand handles the migration of secrets from outside SecretHub to SecretHub.
type ImportCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewImportCommand creates a new ImportCommand.
func NewImportCommand(io ui.IO, newClient newClientFunc) *ImportCommand {
	return &ImportCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ImportCommand) Register(r command.Registerer) {
	clause := r.Command("import", "Import secrets from outside of SecretHub.")
	NewImportDotEnvCommand(cmd.io, cmd.newClient).Register(clause)
}
