package secrethub

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// ReadCommand is a command to read a secret.
type ReadCommand struct {
	io                  ui.IO
	path                api.SecretPath
	useClipboard        bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	outFile             string
	fileMode            filemode.FileMode
	noNewLine           bool
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
func (cmd *ReadCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("read", "Read a secret.")
	clause.Args = cobra.ExactValidArgs(1)
	clause.BoolVarP(&cmd.useClipboard,
		"clip", "c", false,
		fmt.Sprintf(
			"Copy the secret value to the clipboard. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	)
	clause.StringVarP(&cmd.outFile, "out-file", "o", "", "Write the secret value to this file.")
	clause.BoolVarP(&cmd.noNewLine, "no-newline", "n", false, "Do not print a new line after the secret")

	fileModeFlag := clause.VarPF(&cmd.fileMode, "file-mode", "", "Set filemode for the output file. Defaults to 0600 (read and write for current user) and is ignored without the --out-file flag.")
	fileModeFlag.DefValue = "0600"

	command.BindAction(clause, cmd.argumentRegister, cmd.Run)
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

		_, _ = fmt.Fprintf(
			cmd.io.Output(),
			"Copied %s to clipboard. It will be cleared after %s.\n",
			cmd.path,
			units.HumanDuration(cmd.clearClipboardAfter),
		)
	}

	secretData := secret.Data
	if !cmd.noNewLine {
		secretData = posix.AddNewLine(secretData)
	}

	if cmd.outFile != "" {
		err = ioutil.WriteFile(cmd.outFile, secretData, cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.outFile, err)
		}
	}

	if cmd.outFile == "" && !cmd.useClipboard {
		_, _ = fmt.Fprintf(cmd.io.Output(), "%s", string(secretData))
	}

	return nil
}

func (cmd *ReadCommand) argumentRegister(_ *cobra.Command, args []string) error {
	var err error
	cmd.path, err = api.NewSecretPath(args[0])
	if err != nil {
		return err
	}
	return nil
}
