package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-cli/internals/cli"
	"io/ioutil"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-cli/internals/secretspec"
)

// ClearCommand clears the secrets from the system.
type ClearCommand struct {
	in string
	io ui.IO
}

// NewClearCommand creates a new ClearCommand.
func NewClearCommand(io ui.IO) *ClearCommand {
	return &ClearCommand{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ClearCommand) Register(r cli.Registerer) {
	clause := r.Command("clear", "Clear the secrets from your local environment. This reads and parses the secrets.yml file in the current working directory.").Hidden()
	clause.Flags().StringVarP(&cmd.in, "in", "i", "secrets.yml", "The path to a secrets.yml file to read")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run clears the secrets from the system.
func (cmd *ClearCommand) Run() error {
	presenter, err := secretspec.NewPresenter("", true, secretspec.DefaultParsers...)
	if err != nil {
		return err
	}

	_, err = os.Stat(cmd.in)
	if os.IsNotExist(err) {
		return ErrFileNotFound(cmd.in)
	}

	spec, err := ioutil.ReadFile(cmd.in)
	if err != nil {
		return ErrCannotReadFile(cmd.in, err)
	}

	err = presenter.Parse(spec)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Clearing secrets...")

	err = presenter.Clear()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Clear complete! The secrets are no longer available on the system.\n")

	return nil
}
