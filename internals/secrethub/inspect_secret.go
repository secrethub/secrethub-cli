package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
)

// InspectSecretCommand prints out a secret's details.
type InspectSecretCommand struct {
	path          api.SecretPath
	io            ui.IO
	newClient     newClientFunc
	timeFormatter TimeFormatter
}

// NewInspectSecretCommand crates a new InspectSecretCommand
func NewInspectSecretCommand(path api.SecretPath, io ui.IO, newClient newClientFunc) *InspectSecretCommand {
	return &InspectSecretCommand{
		path:          path,
		io:            io,
		newClient:     newClient,
		timeFormatter: NewTimeFormatter(true),
	}
}

// Run prints out a secret's details.
func (cmd *InspectSecretCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	secret, err := client.Secrets().Versions().GetWithoutData(cmd.path.Value())
	if err != nil {
		return err
	}

	versions, err := client.Secrets().Versions().ListWithoutData(cmd.path.Value())
	if err != nil {
		return err
	}

	output, err := cli.PrettyJSON(newSecretOutput(secret.Secret, versions, cmd.timeFormatter))
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Stdout(), output)

	return nil
}

// newSecretOutput returns the JSON output of a secret.
func newSecretOutput(secret *api.Secret, versions []*api.SecretVersion, timeFormatter TimeFormatter) secretOutput {
	out := secretOutput{
		Name:         secret.Name,
		CreatedAt:    timeFormatter.Format(secret.CreatedAt.Local()),
		VersionCount: secret.VersionCount,
		Versions:     make([]secretVersionOutput, len(versions)),
	}

	for i, version := range versions {
		out.Versions[i] = newSecretVersionOutput(version, timeFormatter)
	}

	return out
}

// secretOutput is the printable JSON format of a secret.
type secretOutput struct {
	Name         string
	CreatedAt    string
	VersionCount int
	Versions     []secretVersionOutput
}
