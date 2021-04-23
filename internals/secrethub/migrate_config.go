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
	clause := r.Command("config", "Helper functions to migrate your configuration to use 1Password Secrets Automation syntax.")
	NewMigrateConfigK8sCommand(cmd.io).Register(clause)
}
