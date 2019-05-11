package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secretspec"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	ErrFileNotFound      = errMain.Code("file_not_found").ErrorPref("configuration file `%s` does not exist")
	ErrCannotReadFile    = errMain.Code("cannot_read_file").ErrorPref("cannot read file at %s: %v")
	ErrSecretsNotCleared = errMain.Code("secrets_not_cleared").Error("exiting without having cleared all secrets")
	ErrNoSourcesInSpec   = errMain.Code("no_sources_in_spec").Error("cannot find any sources in the .yml spec file")
)

// SetCommand parses a secret spec file and presents secrets on the system.
type SetCommand struct {
	in        string
	io        ui.IO
	newClient newClientFunc
}

// NewSetCommand creates a new SetCommand.
func NewSetCommand(io ui.IO, newClient newClientFunc) *SetCommand {
	return &SetCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SetCommand) Register(r Registerer) {
	clause := r.Command("set", "Set the secrets in your local environment. This reads and parses the secrets.yml file in the current working directory.").Hidden()
	clause.Flag("in", "The path to a secrets.yml file to read").Short('i').Default("secrets.yml").ExistingFileVar(&cmd.in)

	BindAction(clause, cmd.Run)
}

// Run parses a secret spec file and presents secrets on the system.
func (cmd *SetCommand) Run() error {
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

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	paths := presenter.Sources()
	if len(paths) == 0 {
		return ErrNoSourcesInSpec
	}

	for _, c := range presenter.EmptyConsumables() {
		fmt.Fprintf(cmd.io.Stdout(), "Warning: %s contains no secret declarations.\n", c)
	}

	secrets := make(map[api.SecretPath]api.SecretVersion)
	for path := range paths {
		secret, err := client.Secrets().Versions().GetWithData(path.Value())
		if err != nil {
			return errio.Error(err)
		}
		secrets[path] = *secret
	}

	fmt.Fprintln(cmd.io.Stdout(), "Setting secrets...")

	err = presenter.Set(secrets)
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintln(cmd.io.Stdout(), "Set complete! The secrets are now available on your system.")

	return nil
}
