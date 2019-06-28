package secrethub

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
)

var (
	errCannotWriteToVersion = errMain.Code("cannot_write_version").Error("cannot (over)write a specific secret version, they are append only")
	errEmptySecret          = errMain.Code("cannot_write_empty_secret").Error("secret is empty or contains only whitespace")
)

// WriteCommand is a command to write content to a secret.
type WriteCommand struct {
	io           ui.IO
	path         api.SecretPath
	useClipboard bool
	noTrim       bool
	clipper      clip.Clipper
	newClient    newClientFunc
}

// NewWriteCommand creates a new WriteCommand.
func NewWriteCommand(io ui.IO, newClient newClientFunc) *WriteCommand {
	return &WriteCommand{
		clipper:   clip.NewClipboard(),
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *WriteCommand) Register(r Registerer) {
	clause := r.Command("write", "Write a secret.")
	clause.Arg("secret-path", "The path to the secret (<namespace>/<repo>[/<dir>]/<secret>)").Required().SetValue(&cmd.path)
	clause.Flag("clip", "Use clipboard content as input.").Short('c').BoolVar(&cmd.useClipboard)
	clause.Flag("no-trim", "Do not trim leading and trailing whitespace in the secret.").BoolVar(&cmd.noTrim)

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *WriteCommand) Run() error {
	var err error

	// This error is checked here to fail fast.
	// The error is also checked in the client.
	// Without this check here, the user would be prompted for input when io.Stdin is not piped, but the path is incorrect.
	if cmd.path.HasVersion() {
		return errCannotWriteToVersion
	}

	var data []byte
	if cmd.useClipboard {
		data, err = cmd.clipper.ReadAll()
		if err != nil {
			return err
		}
	} else if cmd.io.Stdin().IsPiped() {
		data, err = ioutil.ReadAll(cmd.io.Stdin())
		if err != nil {
			return ui.ErrReadInput(err)
		}
	} else {
		str, err := ui.AskSecret(cmd.io, "Please type in the value of the secret, followed by an [ENTER]:")
		if err != nil {
			return err
		}
		data = []byte(str)
	}

	if !cmd.noTrim {
		// The data needs to be sanitized and trimmed for whitespace.
		data = bytes.TrimSpace(data)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return errEmptySecret
	}

	_, err = fmt.Fprint(cmd.io.Stdout(), "Writing secret value...\n")
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	version, err := client.Secrets().Write(cmd.path.Value(), data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.io.Stdout(), "Write complete! The given value has been written to %s:%d\n", cmd.path, version.Version)
	if err != nil {
		return err
	}

	return nil
}
