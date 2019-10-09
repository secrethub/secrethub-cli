package secrethub

import (
	"fmt"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

// AuditRepoCommand prints all audit events for a given repository.
type AuditRepoCommand struct {
	io            ui.IO
	path          api.RepoPath
	timeFormatter TimeFormatter
	useTimestamps bool
	newClient     newClientFunc
}

// NewAuditRepoCommand creates a new audit repository command.
func NewAuditRepoCommand(io ui.IO, newClient newClientFunc) *AuditRepoCommand {
	return &AuditRepoCommand{
		io:            io,
		timeFormatter: NewTimeFormatter(false),
		newClient:     newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AuditRepoCommand) Register(r command.Registerer) {
	clause := r.Command("repo", "Show an audit log of all actions on a repository.").Hidden()
	clause.Arg("repo-path", "The repository to audit (<namespace>/<repo>)").SetValue(&cmd.path)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// Run prints all audit events for the given repository.
func (cmd *AuditRepoCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *AuditRepoCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// run prints all audit events for the given repository.
func (cmd *AuditRepoCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	events, err := client.Repos().ListEvents(cmd.path.Value(), nil)
	if err != nil {
		return err
	}

	dirFS, err := client.Dirs().GetTree(cmd.path.GetDirPath().Value(), -1, false)
	if err != nil {
		return err
	}

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)

	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\n", "AUTHOR", "EVENT", "EVENT SUBJECT", "IP ADDRESS", "DATE")

	for i := range events {
		// Loop through list in reverse
		event := events[len(events)-1-i]

		actor, err := getAuditActor(event)
		if err != nil {
			return err
		}

		subject, err := getAuditSubject(event, dirFS)
		if err != nil {
			return err
		}

		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\n",
			actor,
			getEventAction(event),
			subject,
			event.IPAddress,
			cmd.timeFormatter.Format(event.LoggedAt),
		)
	}

	err = tabWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}
