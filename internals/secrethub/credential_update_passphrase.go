package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

type CredentialUpdatePassphraseCommand struct {
	io              ui.IO
	credentialStore CredentialConfig
}

// NewCredentialUpdatePassphraseCommand creates a new CredentialUpdatePassphraseCommand.
func NewCredentialUpdatePassphraseCommand(io ui.IO, credentialStore CredentialConfig) *CredentialUpdatePassphraseCommand {
	return &CredentialUpdatePassphraseCommand{
		io:              io,
		credentialStore: credentialStore,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CredentialUpdatePassphraseCommand) Register(r cli.Registerer) {
	clause := r.Command("update-passphrase", "Update the passphrase of your local key credential file.")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run upgrades the configuration in the profile directory to the new version.
func (cmd *CredentialUpdatePassphraseCommand) Run() error {
	if !cmd.credentialStore.ConfigDir().Credential().Exists() {
		fmt.Println("No credentials. Nothing to do.")
		return nil
	}
	// Run command
	confirmed, err := ui.AskYesNo(
		cmd.io,
		fmt.Sprintf(
			"Do you want to update the passphrase of your local key credential stored at %s?",
			cmd.credentialStore.ConfigDir(),
		),
		ui.DefaultYes,
	)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Fprintln(cmd.io.Output(), "Aborting.")
		return nil
	}

	credential, err := cmd.credentialStore.Import()
	if err != nil {
		return err
	}

	passphrase, err := ui.AskPassphrase(cmd.io, "Please enter a passphrase to (re)encrypt your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
	if err != nil {
		return err
	}
	if passphrase != "" {
		credential = credential.Passphrase(credentials.FromString(passphrase))
	}
	exportedCredential, err := credential.Export()
	if err != nil {
		return err
	}

	err = cmd.credentialStore.ConfigDir().Credential().Write(exportedCredential)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Successfully updated passphrase!")

	return nil
}
