package demo

//import (
//	"github.com/secrethub/demo-app/cli"
//
//	"github.com/secrethub/secrethub-cli/internals/cli/ui"
//	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
//)
//
//// Command is a command to run the secrethub example app.
//type Command struct {
//	io        ui.IO
//	newClient newClientFunc
//}
//
//// NewCommand creates a new example app command.
//func NewCommand(io ui.IO, newClient newClientFunc) *Command {
//	return &Command{
//		io:        io,
//		newClient: newClient,
//	}
//}
//
//// Register registers the command, arguments and flags on the provided Registerer.
//func (cmd *Command) Register(r command.Registerer) {
//	clause := r.CreateCommand("demo", "Manage the demo application.")
//	clause.Hidden()
//
//	NewInitCommand(cmd.io, cmd.newClient).Register(clause)
//	cli.NewServeCommand(cmd.io).Register(clause)
//}
