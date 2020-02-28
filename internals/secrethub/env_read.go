package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// EnvReadCommand is a command to read the value of a single environment variable.
type EnvReadCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewEnvReadCommand creates a new EnvReadCommand.
func NewEnvReadCommand(io ui.IO, newClient newClientFunc) *EnvReadCommand {
	return &EnvReadCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
func (cmd *EnvReadCommand) Register(r command.Registerer) {
	clause := r.Command("read", "Read the value of a single environment variable.")

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *EnvReadCommand) Run() error {
	return nil
}
