package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

type MigrateConfigCommand struct {
	io ui.IO
}

func NewMigrateConfigCommand(io ui.IO) *MigrateConfigCommand {
	return &MigrateConfigCommand{
		io: io,
	}
}

func (cmd *MigrateConfigCommand) Register(r cli.Registerer) {
	clause := r.Command("config", "Helper functions to migrate your configuration code to make it work with 1Password.\n\nNote: These commands should be considered best effort and the output should be carefully tested and reviewed before using in production.")
	NewMigrateConfigK8sCommand(cmd.io).Register(clause)
	NewMigrateConfigReferencesCommand(cmd.io).Register(clause)
	NewMigrateConfigTemplatesCommand(cmd.io).Register(clause)
	NewMigrateConfigEnvfileCommand(cmd.io).Register(clause)
}
