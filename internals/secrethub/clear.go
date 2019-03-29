package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secretspec"
	"github.com/secrethub/secrethub-go/internals/errio"
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
func (cmd *ClearCommand) Register(r Registerer) {
	clause := r.Command("clear", "Clear the secrets from your local environment. This reads and parses the secrets.yml file in the current working directory.")
	clause.Flag("in", "The path to a secrets.yml file to read").Short('i').Default("secrets.yml").ExistingFileVar(&cmd.in)

	BindAction(clause, cmd.Run)
}

// Run clears the secrets from the system.
func (cmd *ClearCommand) Run() error {
	presenter, err := secretspec.NewPresenter("", true, secretspec.DefaultParsers...)
	if err != nil {
		return errio.Error(err)
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
		return errio.Error(err)
	}

	fmt.Fprintln(cmd.io.Stdout(), "Clearing secrets...")

	err = presenter.Clear()
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintf(cmd.io.Stdout(), "Clear complete! The secrets are no longer available on the system.\n")

	return nil
}
