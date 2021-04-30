package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/docker/go-units"
)

// ReadCommand is a command to read a secret.
type ReadCommand struct {
	io            ui.IO
	path          api.SecretPath
	useClipboard  bool
	outFile       string
	fileMode      filemode.FileMode
	noNewLine     bool
	newClient     newClientFunc
	writeFileFunc func(filename string, data []byte, perm os.FileMode) error
	clipWriter    ClipboardWriter
}

// NewReadCommand creates a new ReadCommand.
func NewReadCommand(io ui.IO, newClient newClientFunc) *ReadCommand {
	return &ReadCommand{
		clipWriter: &ClipboardWriterAutoClear{
			clipper: clip.NewClipboard(),
		},
		io:            io,
		newClient:     newClient,
		writeFileFunc: ioutil.WriteFile,
		fileMode:      filemode.New(0600),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ReadCommand) Register(r cli.Registerer) {
	clause := r.Command("read", "Read a secret.")

	clause.Flags().BoolVarP(&cmd.useClipboard,
		"clip", "c", false,
		fmt.Sprintf(
			"Copy the secret value to the clipboard. The clipboard is automatically cleared after %s.",
			units.HumanDuration(clearClipboardAfter),
		),
	)
	clause.Flags().StringVarP(&cmd.outFile, "out-file", "o", "", "Write the secret value to this file.")
	clause.Flags().BoolVarP(&cmd.noNewLine, "no-newline", "n", false, "Do not print a new line after the secret")
	clause.Flags().VarPF(&cmd.fileMode, "file-mode", "", "Set filemode for the output file. It is ignored without the --out-file flag.")

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.Argument{{Value: &cmd.path, Name: "path", Placeholder: secretPathOptionalVersionPlaceHolder, Required: true, Description: "The path to the secret."}})
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
		err = cmd.clipWriter.Write(secret.Data)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(
			cmd.io.Output(),
			"Copied %s to clipboard. It will be cleared after %s.\n",
			cmd.path,
			units.HumanDuration(clearClipboardAfter),
		)
	}

	secretData := secret.Data
	if !cmd.noNewLine {
		secretData = posix.AddNewLine(secretData)
	}

	if cmd.outFile != "" {
		err = cmd.writeFileFunc(cmd.outFile, secretData, cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.outFile, err)
		}
	}

	if cmd.outFile == "" && !cmd.useClipboard {
		_, _ = fmt.Fprintf(cmd.io.Output(), "%s", string(secretData))
	}

	return nil
}
