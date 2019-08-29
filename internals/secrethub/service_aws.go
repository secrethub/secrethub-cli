package secrethub

import "github.com/secrethub/secrethub-cli/internals/cli/ui"

// ServiceAWSCommand handles AWS services.
type ServiceAWSCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewServiceAWSCommand creates a new ServiceAWSCommand.
func NewServiceAWSCommand(io ui.IO, newClient newClientFunc) *ServiceAWSCommand {
	return &ServiceAWSCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ServiceAWSCommand) Register(r Registerer) {
	clause := r.Command("aws", "Manage AWS service accounts.")
	NewServiceAWSInitCommand(cmd.io, cmd.newClient).Register(clause)
	NewServiceAWSLsCommand(cmd.io, cmd.newClient).Register(clause)
}
