package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// EnvReadCommand is a command to read the value of a single environment variable.
type EnvReadCommand struct {
	io          ui.IO
	newClient   newClientFunc
	environment *environment
	key         cli.StringValue
}

// NewEnvReadCommand creates a new EnvReadCommand.
func NewEnvReadCommand(io ui.IO, newClient newClientFunc) *EnvReadCommand {
	return &EnvReadCommand{
		io:          io,
		newClient:   newClient,
		environment: newEnvironment(io, newClient),
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
func (cmd *EnvReadCommand) Register(r cli.Registerer) {
	clause := r.Command("read", "[BETA] Read the value of a single environment variable.")
	clause.HelpLong("This command is hidden because it is still in beta. Future versions may break.")

	cmd.environment.register(clause)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.Argument{
		{Value: &cmd.key, Name: "key", Required: false, Description: "the key of the environment variable to read."},
	})
}

// Run executes the command.
func (cmd *EnvReadCommand) Run() error {
	env, err := cmd.environment.env()
	if err != nil {
		return err
	}

	value, found := env[cmd.key.Value]
	if !found {
		return fmt.Errorf("no environment variable with that key is set")
	}

	secretReader := newSecretReader(cmd.newClient)

	res, err := value.resolve(secretReader)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), res)

	return nil
}
