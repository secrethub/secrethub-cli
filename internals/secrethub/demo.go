package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// DemoCommand handles operations on the demo application.
type DemoCommand struct {
	io ui.IO
	newClient newClientFunc
}

// NewDemoCommand creates a new DemoCommand.
func NewDemoCommand(io ui.IO, newClient newClientFunc) *DemoCommand {
	return &DemoCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *DemoCommand) Register(r Registerer) {
	clause := r.Command("demo", "Manage the demo application.")

	NewDemoInitCommand(cmd.io, cmd.newClient).Register(clause)
}
