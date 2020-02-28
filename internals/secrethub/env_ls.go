package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// EnvListCommand is a command to list all environment variable keys set in the process of `secrethub run`.
type EnvListCommand struct {
	io          ui.IO
	environment *environment
}

// NewEnvListCommand creates a new EnvListCommand.
func NewEnvListCommand(io ui.IO) *EnvListCommand {
	return &EnvListCommand{
		io:          io,
		environment: newEnvironment(io),
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
func (cmd *EnvListCommand) Register(r command.Registerer) {
	clause := r.Command("ls", "Read the value of a single environment variable.")
	clause.Alias("list")

	cmd.environment.register(clause)

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *EnvListCommand) Run() error {
	env, err := cmd.environment.env()
	if err != nil {
		return err
	}

	for key, value := range env {
		// For now only environment variables in which a secret is loaded are printed.
		// TODO: Make this behavior configurable.
		if value.containsSecret() {
			fmt.Fprintln(cmd.io.Stdout(), key)
		}
	}

	return nil
}