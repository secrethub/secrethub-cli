package secrethub

import (
	"fmt"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// ConfigCommand handles operations on the SecretHub configuration.
type ConfigCommand struct {
	io              ui.IO
	credentialStore CredentialStore
}

// NewConfigCommand creates a new ConfigCommand.
func NewConfigCommand(io ui.IO, store CredentialStore) *ConfigCommand {
	return &ConfigCommand{
		io:              io,
		credentialStore: store,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ConfigCommand) Register(r Registerer) {
	clause := r.Command("config", "Manage your local .secrethub configuration file.")
	NewConfigUpgradeCommand(cmd.io, cmd.credentialStore).Register(clause)
}

// ConfigUpgradeCommand handles upgrading the configuration in the profile directory.
type ConfigUpgradeCommand struct {
	io              ui.IO
	credentialStore CredentialStore
}

// NewConfigUpgradeCommand creates a new ConfigUpgradeCommand.
func NewConfigUpgradeCommand(io ui.IO, credentialStore CredentialStore) *ConfigUpgradeCommand {
	return &ConfigUpgradeCommand{
		io:              io,
		credentialStore: credentialStore,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ConfigUpgradeCommand) Register(r Registerer) {
	clause := r.Command("upgrade", "Upgrade your .secrethub configuration directory. This can be useful to migrate to a newer version of the configuration files.")

	BindAction(clause, cmd.Run)
}

// Run upgrades the configuration in the profile directory to the new version.
func (cmd *ConfigUpgradeCommand) Run() error {
	profileDir, err := cmd.credentialStore.NewProfileDir()
	if err != nil {
		return errio.Error(err)
	}

	// Run command
	confirmed, err := ui.AskYesNo(
		cmd.io,
		fmt.Sprintf(
			"This upgrades your SecretHub account credential to the latest format. "+
				"Are you sure you wish upgrade the configuration stored at %s?",
			profileDir,
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

	var cleanupFiles []string
	if profileDir.IsOldConfiguration() {
		// Ensure files are removed upon successful upgrade
		cleanupFiles = append(cleanupFiles, profileDir.oldConfigFile())

		config, err := LoadConfig(cmd.io, profileDir.oldConfigFile())
		if err != nil {
			return errio.Error(err)
		}

		if config.Type == ConfigUserType {
			cleanupFiles = append(cleanupFiles, config.User.KeyFile)
		}
	}

	credential, err := cmd.credentialStore.Get()
	if err != nil {
		return errio.Error(err)
	}

	passphrase, err := ui.AskPassphrase(cmd.io, "Please enter a passphrase to (re)encrypt your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
	if err != nil {
		return errio.Error(err)
	}

	cmd.credentialStore.SetPassphrase(passphrase)
	cmd.credentialStore.Set(credential)
	err = cmd.credentialStore.Save()
	if err != nil {
		return errio.Error(err)
	}

	// Remove old files
	for _, file := range cleanupFiles {
		err = os.Remove(file)
		if err != nil {
			return errio.Error(err)
		}
	}

	return nil
}
