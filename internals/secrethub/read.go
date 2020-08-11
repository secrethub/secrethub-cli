package secrethub

import (
	"fmt"
	"github.com/docker/go-units"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"io/ioutil"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/spf13/cobra"
)

// ReadCommand is a command to read a secret.
type ReadCommand struct {
	io                  ui.IO
	path                api.SecretPath
	useClipboard        *bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	outFile             *string
	fileMode            filemode.FileMode
	noNewLine           *bool
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
	cmd.argumentConstraint(clause.Command)
	cmd.useClipboard = clause.Flags().BoolP(
		"clip", "c", false,
		fmt.Sprintf(
			"Copy the secret value to the clipboard. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	)
	cmd.outFile = clause.Flags().StringP("out-file", "o", "",  "Write the secret value to this file.")
	cmd.fileMode = ("file-mode", "Set filemode for the output file. Defaults to 0600 (read and write for current user) and is ignored without the --out-file flag.").Default("0600").SetValue(&cmd.fileMode)
	cmd.noNewLine = clause.Flags().BoolP("no-newline", "n", false, "Do not print a new line after the secret")
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

	if *cmd.useClipboard {
		err = WriteClipboardAutoClear(secret.Data, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			cmd.io.Output(),
			"Copied %s to clipboard. It will be cleared after %s.\n",
			cmd.path,
			units.HumanDuration(cmd.clearClipboardAfter),
		)
	}

	secretData := secret.Data
	if !*cmd.noNewLine {
		secretData = posix.AddNewLine(secretData)
	}

	if *cmd.outFile != "" {
		err = ioutil.WriteFile(*cmd.outFile, secretData, cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.outFile, err)
		}
	}

	if *cmd.outFile == "" && !*cmd.useClipboard {
		fmt.Fprintf(cmd.io.Output(), "%s", string(secretData))
	}

	return nil
}

func (cmd *ReadCommand) argumentConstraint(c *cobra.Command) {
	c.Args = cobra.ExactValidArgs(1)
}

func (cmd *ReadCommand) argumentRegister(c *cobra.Command, args []string) error {
	var err error
	cmd.path, err = api.NewSecretPath(args[0])
	if err != nil {
		return err
	}
	return nil
}
