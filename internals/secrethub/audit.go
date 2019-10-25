package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

// AuditCommand is a command to audit a repo or a secret.
type AuditCommand struct {
	io            ui.IO
	path          api.Path
	useTimestamps bool
	newClient     newClientFunc
	perPage       int
}

// NewAuditCommand creates a new audit command.
func NewAuditCommand(io ui.IO, newClient newClientFunc) *AuditCommand {
	return &AuditCommand{
		io:        io,
		newClient: newClient,
		perPage:   10,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AuditCommand) Register(r command.Registerer) {
	clause := r.Command("audit", "Show the audit log.")
	clause.Arg("repo-path or secret-path", "Path to the repository or the secret to audit "+repoPathPlaceHolder+" or "+secretPathPlaceHolder).SetValue(&cmd.path)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// Run prints all audit events for the given repository or secret.
func (cmd *AuditCommand) Run() error {
	repoPath, err := cmd.path.ToRepoPath()
	if err == nil {
		auditRepoCommand := AuditRepoCommand{
			io:            cmd.io,
			path:          repoPath,
			useTimestamps: cmd.useTimestamps,
			newClient:     cmd.newClient,
			perPage:       cmd.perPage,
		}
		return auditRepoCommand.Run()
	}

	secretPath, err := cmd.path.ToSecretPath()
	if err == nil {
		auditSecretCommand := AuditSecretCommand{
			io:            cmd.io,
			path:          secretPath,
			useTimestamps: cmd.useTimestamps,
			newClient:     cmd.newClient,
			perPage:       cmd.perPage,
		}
		return auditSecretCommand.Run()
	}

	return ErrNoValidRepoOrSecretPath
}
