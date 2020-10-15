package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// EnvListCommand is a command to list all environment variable keys set in the process of `secrethub run`.
type EnvListCommand struct {
	io          ui.IO
	environment *environment
}

// NewEnvListCommand creates a new EnvListCommand.
func NewEnvListCommand(io ui.IO, newClient newClientFunc) *EnvListCommand {
	return &EnvListCommand{
		io:          io,
		environment: newEnvironment(io, newClient),
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
func (cmd *EnvListCommand) Register(r cli.Registerer) {
	clause := r.Command("ls", "[BETA] List environment variable names that will be populated with secrets.")
	clause.HelpLong("This command is hidden because it is still in beta. Future versions may break.")
	clause.Alias("list")

	cmd.environment.register(clause)

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil, nil)
}

// Run executes the command.
func (cmd *EnvListCommand) Run() error {
	env, err := cmd.environment.env()
	if err != nil {
		return err
	}

	for key, value := range env {
		// For now only environment variables in which a secret is loaded are printed.
		// TODO: Make this behavior configurable.
		if value.containsSecret() {
			fmt.Fprintln(cmd.io.Output(), key)
		}
	}

	return nil
}
