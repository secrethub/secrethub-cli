package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// OrgCommand handles operations on organizations.
type OrgCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewOrgCommand creates a new OrgCommand.
func NewOrgCommand(io ui.IO, newClient newClientFunc) *OrgCommand {
	return &OrgCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *OrgCommand) Register(r command.Registerer) {
	clause := r.Command("org", "Manage shared organization workspaces.")
	clause.Alias("organization")
	clause.Alias("organisation")
	clause.Alias("orgs")
	clause.Alias("organizations")
	clause.Alias("organisations")
	NewOrgInitCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgInspectCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgInviteCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgPurchaseCommand(cmd.io).Register(clause)
	NewOrgListUsersCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgLsCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgRevokeCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgRmCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgSetRoleCommand(cmd.io, cmd.newClient).Register(clause)
}
