package secrethub

import (
	"fmt"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-go/internals/api"
)

// AuditRepoCommand prints all audit events for a given repository.
type AuditRepoCommand struct {
	io            ui.IO
	path          api.RepoPath
	timeFormatter TimeFormatter
	useTimestamps bool
	newClient     newClientFunc
	perPage       int
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

	dirFS, err := client.Dirs().GetTree(cmd.path.GetDirPath().Value(), -1, false)
	if err != nil {
		return err
	}

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)

	header := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", "AUTHOR", "EVENT", "EVENT SUBJECT", "IP ADDRESS", "DATE")
	fmt.Fprintf(tabWriter, header)

	// interactive mode is assumed, except when output is piped.
	interactive := !cmd.io.Stdout().IsPiped()

	iter := client.Repos().EventIterator(cmd.path.Value(), nil)
	i := 0
	for {
		i++
		event, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

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

		if interactive && i == cmd.perPage {
			err = tabWriter.Flush()
			if err != nil {
				return err
			}
			i = 0
			fmt.Fprintln(cmd.io.Stdout(), "Press <ENTER> to show more results. Press <CTRL+C> to exit.")

			// wait for <ENTER> to continue.
			_, err := ui.Readln(cmd.io.Stdin())
			if err != nil {
				return err
			}
			fmt.Fprintf(tabWriter, header)
		}
	}

	err = tabWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}
