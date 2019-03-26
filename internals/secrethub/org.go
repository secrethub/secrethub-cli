package secrethub

import "github.com/keylockerbv/secrethub-cli/internals/cli/ui"

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
func (cmd *OrgCommand) Register(r Registerer) {
	clause := r.Command("org", "Manage a SecretHub organization.")
	NewOrgInitCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgInspectCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgInviteCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgListUsersCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgLsCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgRevokeCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgRmCommand(cmd.io, cmd.newClient).Register(clause)
	NewOrgSetRoleCommand(cmd.io, cmd.newClient).Register(clause)
}
