package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	units "github.com/docker/go-units"
	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/clip"
	"github.com/keylockerbv/secrethub-cli/pkg/filemode"
	"github.com/keylockerbv/secrethub-cli/pkg/posix"
	"github.com/keylockerbv/secrethub-cli/pkg/injection"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// InjectCommand is a command to read a secret.
type InjectCommand struct {
	file                string
	fileMode            filemode.FileMode
	force               bool
	io                  ui.IO
	useClipboard        bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	newClient           newClientFunc
}

// NewInjectCommand creates a new InjectCommand.
func NewInjectCommand(io ui.IO, newClient newClientFunc) *InjectCommand {
	return &InjectCommand{
		clipper:             clip.Clipboard,
		clearClipboardAfter: defaultClearClipboardAfter,
		io:                  io,
		newClient:           newClient,
	}
}

// Register adds a CommandClause and it's args and flags to a cli.App.
// Register adds args and flags.
func (cmd *InjectCommand) Register(r Registerer) {
	clause := r.Command("inject", "Read a template from stdin and write to stdout with secrets injected into the template.")
	clause.Flag(
		"clip",
		fmt.Sprintf(
			"Copy the injected template to the clipboard instead of stdout. The clipboard is automatically cleared after %s.",
			units.HumanDuration(cmd.clearClipboardAfter),
		),
	).Short('c').BoolVar(&cmd.useClipboard)
	clause.Flag("file", "Write the injected template to a file instead of stdout.").StringVar(&cmd.file)
	clause.Flag("file-mode", "Set filemode for the file if it does not yet exist. Defaults to 0600 (read and write for current user) and is ignored without the --file flag.").Default("0600").SetValue(&cmd.fileMode)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *InjectCommand) Run() error {
	if cmd.useClipboard && cmd.file != "" {
		return ErrFlagsConflict("--clip and --file")
	}

	var err error

	if !cmd.io.Stdin().IsPiped() {
		return ErrNoDataOnStdin
	}

	raw, err := ioutil.ReadAll(cmd.io.Stdin())
	if err != nil {
		return errio.Error(err)
	}

	tpl, err := injection.NewTemplate(string(raw))
	if err != nil {
		return errio.Error(err)
	}

	secrets := make(map[api.SecretPath][]byte)

	var client secrethub.Client
	if len(tpl.Secrets) > 0 {
		client, err = cmd.newClient()
		if err != nil {
			return errio.Error(err)
		}
	}

	for _, path := range tpl.Secrets {
		secret, err := client.Secrets().Versions().GetWithData(path.Value())
		if err != nil {
			return errio.Error(err)
		}
		secrets[path] = secret.Data
	}

	injected, err := tpl.Inject(secrets)
	if err != nil {
		return errio.Error(err)
	}

	out := []byte(injected)
	if cmd.useClipboard {
		err = WriteClipboardAutoClear(out, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return errio.Error(err)
		}

		fmt.Fprintln(cmd.io.Stdout(), fmt.Sprintf("Copied injected template to clipboard. It will be cleared after %s.", units.HumanDuration(cmd.clearClipboardAfter)))
	} else if cmd.file != "" {
		_, err := os.Stat(cmd.file)
		if err == nil && !cmd.force {
			if cmd.io.Stdout().IsPiped() {
				return ErrFileAlreadyExists
			}

			confirmed, err := ui.AskYesNo(
				cmd.io,
				fmt.Sprintf(
					"File %s already exists, overwrite it?",
					cmd.file,
				),
				ui.DefaultNo,
			)
			if err != nil {
				return errio.Error(err)
			}

			if !confirmed {
				fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
				return nil
			}
		}

		err = ioutil.WriteFile(cmd.file, posix.AddNewLine(out), cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.file, err)
		}

		absPath, err := filepath.Abs(cmd.file)
		if err != nil {
			return ErrCannotWrite(err)
		}

		fmt.Fprintf(cmd.io.Stdout(), "%s\n", absPath)
	} else {
		fmt.Fprintf(cmd.io.Stdout(), "%s", posix.AddNewLine(out))
	}

	return nil
}
