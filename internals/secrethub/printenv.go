package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// PrintEnvCommand prints out debug statements about all environment variables.
type PrintEnvCommand struct {
	app     *cli.App
	io      ui.IO
	verbose bool
}

// NewPrintEnvCommand creates a new PrintEnvCommand.
func NewPrintEnvCommand(app *cli.App, io ui.IO) *PrintEnvCommand {
	return &PrintEnvCommand{
		app: app,
		io:  io,
	}
}

// Run prints out debug statements about all environment variables.
func (cmd *PrintEnvCommand) Run() error {
	err := cmd.app.PrintEnv(cmd.io.Stdout(), cmd.verbose)
	if err != nil {
		return err
	}
	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *PrintEnvCommand) Register(r command.Registerer) {
	clause := r.Command("printenv", "Print environment variables.")
	clause.Flag("verbose", "Show all possible environment variables.").Short('v').BoolVar(&cmd.verbose)

	command.BindAction(clause, cmd.Run)
}
