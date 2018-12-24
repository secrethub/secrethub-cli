package secrethub

import "github.com/keylockerbv/secrethub-cli/pkg/ui"

// RepoCommand handles operations on repositories.
type RepoCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewRepoCommand creates a new RepoCommand.
func NewRepoCommand(io ui.IO, newClient newClientFunc) *RepoCommand {
	return &RepoCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *RepoCommand) Register(r Registerer) {
	clause := r.Command("repo", "Manage repositories.")
	NewRepoInitCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoInspectCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoInviteCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoExportCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoLSCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoRevokeCommand(cmd.io, cmd.newClient).Register(clause)
	NewRepoRmCommand(cmd.io, cmd.newClient).Register(clause)
}
