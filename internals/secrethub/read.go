package secrethub

import (
	"fmt"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"

	units "github.com/docker/go-units"
)

// ReadCommand is a command to read a secret.
type ReadCommand struct {
	io                  ui.IO
	path                api.SecretPath
	useClipboard        bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	newClient           newClientFunc
}

// NewReadCommand creates a new ReadCommand.
func NewReadCommand(io ui.IO, newClient newClientFunc) *ReadCommand {
	return &ReadCommand{
		clipper:             clip.NewClipboard(),
		clearClipboardAfter: defaultClearClipboardAfter,
		io:                  io,
		newClient:           newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ReadCommand) Register(r Registerer) {
	clause := r.Command("read", "Read a secret.")
	clause.Arg("secret-path", "The path to the secret (<namespace>/<repo>[/<dir>]/<secret>)").Required().SetValue(&cmd.path)
	clause.Flag(
		"clip",
		fmt.Sprintf(
			"Copy the secret value to the clipboard. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	).Short('c').BoolVar(&cmd.useClipboard)

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *ReadCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	secret, err := client.Secrets().Versions().GetWithData(cmd.path.Value())
	if err != nil {
		return err
	}

	if cmd.useClipboard {
		err = WriteClipboardAutoClear(secret.Data, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			cmd.io.Stdout(),
			"Copied %s to clipboard. It will be cleared after %s.\n",
			cmd.path,
			units.HumanDuration(cmd.clearClipboardAfter),
		)
	} else {
		fmt.Fprintf(cmd.io.Stdout(), "%s", string(posix.AddNewLine(secret.Data)))
	}

	return nil
}
