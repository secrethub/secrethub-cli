package secrethub

import (
	"fmt"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// AuditSecretCommand prints all audit events for a given secret.
type AuditSecretCommand struct {
	io            ui.IO
	path          api.SecretPath
	timeFormatter TimeFormatter
	useTimestamps bool
	newClient     newClientFunc
}

// NewAuditSecretCommand creates a new audit repository command.
func NewAuditSecretCommand(io ui.IO, newClient newClientFunc) *AuditSecretCommand {
	return &AuditSecretCommand{
		io:            io,
		timeFormatter: NewTimeFormatter(false),
		newClient:     newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AuditSecretCommand) Register(r Registerer) {
	clause := r.Command("secret", "Show an audit log of all actions on a secret.").Hidden()
	clause.Arg("secret-path", "The path to the secret to audit (<namespace>/<repo>[/<dir>]/<secret>)").SetValue(&cmd.path)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	BindAction(clause, cmd.Run)
}

// Run prints all audit events for the given secret.
func (cmd *AuditSecretCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *AuditSecretCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// run prints all audit events for the given secret.
func (cmd *AuditSecretCommand) run() error {

	if cmd.path.HasVersion() {
		return ErrCannotAuditSecretVersion
	}

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	events, err := client.Secrets().ListEvents(cmd.path.Value(), nil)
	if err == api.ErrSecretNotFound {
		// Check if we're attempting to audit a dir.
		_, err = client.Dirs().GetTree(api.DirPath(cmd.path).Value(), 1, false)
		if err == nil {
			return ErrCannotAuditDir
		}
		return api.ErrSecretNotFound
	}

	if err != nil {
		return errio.Error(err)
	}

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)

	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n", "AUTHOR", "EVENT", "IP ADDRESS", "DATE")

	for i := range events {
		// Loop through list in reverse
		event := events[len(events)-1-i]

		actor, err := getAuditActor(event)
		if err != nil {
			return errio.Error(err)
		}

		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n",
			actor,
			getEventAction(event),
			event.IPAddress,
			cmd.timeFormatter.Format(event.LoggedAt),
		)
	}

	err = tabWriter.Flush()
	if err != nil {
		return errio.Error(err)
	}

	return nil
}
