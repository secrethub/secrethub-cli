package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// EnvReadCommand is a command to read the value of a single environment variable.
type EnvReadCommand struct {
	io          ui.IO
	newClient   newClientFunc
	environment *environment
	key         string
}

// NewEnvReadCommand creates a new EnvReadCommand.
func NewEnvReadCommand(io ui.IO, newClient newClientFunc) *EnvReadCommand {
	return &EnvReadCommand{
		io:          io,
		newClient:   newClient,
		environment: newEnvironment(io),
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
func (cmd *EnvReadCommand) Register(r command.Registerer) {
	clause := r.Command("read", "Read the value of a single environment variable.")
	clause.Arg("key", "the key of the environment variable to read").StringVar(&cmd.key)

	cmd.environment.register(clause)

	command.BindAction(clause, cmd.Run)
}

// Run executes the command.
func (cmd *EnvReadCommand) Run() error {
	env, err := cmd.environment.env()
	if err != nil {
		return err
	}

	value, found := env[cmd.key]
	if !found {
		return fmt.Errorf("no environment variable with that key is set")
	}

	secretReader := newSecretReader(cmd.newClient)

	res, err := value.resolve(secretReader)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Stdout(), res)

	return nil
}
