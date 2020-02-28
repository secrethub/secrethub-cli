package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// EnvCommand handles operations regarding environment variables.
type EnvCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewEnvCommand creates a new EnvCommand.
func NewEnvCommand(io ui.IO, newClient newClientFunc) *EnvCommand {
	return &EnvCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *EnvCommand) Register(r command.Registerer) {
	clause := r.Command("env", "Manage environment variables.").Hidden() // The command is hidden, because it's still in beta.
	NewEnvReadCommand(cmd.io, cmd.newClient).Register(clause)
}
