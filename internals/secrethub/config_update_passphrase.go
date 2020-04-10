package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

type ConfigUpdatePassphraseCommand struct {
	io              ui.IO
	credentialStore CredentialConfig
}

// NewConfigUpdatePassphraseCommand creates a new ConfigUpdatePassphraseCommand.
func NewConfigUpdatePassphraseCommand(io ui.IO, credentialStore CredentialConfig) *ConfigUpdatePassphraseCommand {
	return &ConfigUpdatePassphraseCommand{
		io:              io,
		credentialStore: credentialStore,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ConfigUpdatePassphraseCommand) Register(r command.Registerer) {
	clause := r.Command("update-passphrase", "Update the passphrase of your local key credential file.")

	command.BindAction(clause, cmd.Run)
}

// Run upgrades the configuration in the profile directory to the new version.
func (cmd *ConfigUpdatePassphraseCommand) Run() error {
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
		fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
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

	fmt.Fprintln(cmd.io.Stdout(), "Successfully updated passphrase!")

	return nil
}
