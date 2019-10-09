package demo_app

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/demo-app/app"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

type ServeCommand struct {
	io ui.IO

	host string
	port int
}

func NewServeCommand(io ui.IO) *ServeCommand {
	return &ServeCommand{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServeCommand) Register(r command.Registerer) {
	clause := r.Command("serve", "Runs the secrethub example by serving a web page.")

	clause.Flag("host", "The host to serve the webpage on").Short('h').Default("127.0.0.1").StringVar(&cmd.host)
	clause.Flag("port", "The port to serve the webpage on").Default("8080").IntVar(&cmd.port)

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *ServeCommand) Run() error {
	fmt.Fprintf(cmd.io.Stdout(), "Serving example app on http://%s:%d\n", cmd.host, cmd.port)
	return app.NewServer(cmd.host, cmd.port).Serve()
}
