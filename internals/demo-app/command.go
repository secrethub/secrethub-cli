package demo_app

import (
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
	clause := r.Command("demo", "Runs the secrethub demo app as used in different guides.")

	NewServeCommand(cmd.io).Register(clause)
}
