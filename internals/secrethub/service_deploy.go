package secrethub

import "github.com/keylockerbv/secrethub-cli/internals/cli/ui"

// ServiceDeployCommand handles deploying a service.
type ServiceDeployCommand struct {
	io ui.IO
}

// NewServiceDeployCommand creates a new ServiceDeployCommand.
func NewServiceDeployCommand(io ui.IO) *ServiceDeployCommand {
	return &ServiceDeployCommand{
		io: io,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ServiceDeployCommand) Register(r Registerer) {
	clause := r.Command("deploy", "Deploy a service account to a destination.")
	NewServiceDeployWinRmCommand(cmd.io).Register(clause)
}
