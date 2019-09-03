package example_app

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// Command is a command to run the secrethub example app.
type Command struct {
	io ui.IO
}

// NewCommand creates a new example app command.
func NewCommand(io ui.IO) *Command {
	return &Command{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *Command) Register(r command.Registerer) {
	clause := r.Command("example-app", "Runs the secrethub example app as used in different guides.")

	NewServeCommand(cmd.io).Register(clause)

}

// Run handles the command with the options as specified in the command.
func (cmd *Command) Run() error {
	fmt.Println("test")

	return nil
}
