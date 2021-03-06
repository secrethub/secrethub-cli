package secrethub

import (
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// PrintEnvCommand prints out debug statements about all environment variables.
type PrintEnvCommand struct {
	app     *cli.App
	io      ui.IO
	osEnv   func() []string
	verbose bool
}

// NewPrintEnvCommand creates a new PrintEnvCommand.
func NewPrintEnvCommand(app *cli.App, io ui.IO) *PrintEnvCommand {
	return &PrintEnvCommand{
		app:   app,
		io:    io,
		osEnv: os.Environ,
	}
}

// Run prints out debug statements about all environment variables.
func (cmd *PrintEnvCommand) Run() error {
	err := cmd.app.PrintEnv(cmd.io.Output(), cmd.verbose, cmd.osEnv)
	if err != nil {
		return err
	}
	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *PrintEnvCommand) Register(r cli.Registerer) {
	clause := r.Command("printenv", "Print environment variables.")
	clause.Flags().BoolVarP(&cmd.verbose, "verbose", "v", false, "Show all possible environment variables.")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}
