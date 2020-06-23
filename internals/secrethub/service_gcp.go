package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// ServiceGCPCommand handles GCP services.
type ServiceGCPCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewServiceGCPCommand creates a new ServiceGCPCommand.
func NewServiceGCPCommand(io ui.IO, newClient newClientFunc) *ServiceGCPCommand {
	return &ServiceGCPCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ServiceGCPCommand) Register(r command.Registerer) {
	clause := r.Command("gcp", "Manage GCP service accounts.").Hidden()
	NewServiceGCPInitCommand(cmd.io, cmd.newClient).Register(clause)
	NewServiceGCPLsCommand(cmd.io, cmd.newClient).Register(clause)
	NewServiceGCPLinkCommand(cmd.io, cmd.newClient).Register(clause)
	NewServiceGCPListLinksCommand(cmd.io, cmd.newClient).Register(clause)
	NewServiceGCPDeleteLinkCommand(cmd.io, cmd.newClient).Register(clause)
}
