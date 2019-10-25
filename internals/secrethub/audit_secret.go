package secrethub

import (
	"fmt"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-go/internals/api"
)

// AuditSecretCommand prints all audit events for a given secret.
type AuditSecretCommand struct {
	io            ui.IO
	path          api.SecretPath
	timeFormatter TimeFormatter
	useTimestamps bool
	newClient     newClientFunc
	perPage       int
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
		return err
	}

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)

	header := fmt.Sprintf("%s\t%s\t%s\t%s\n", "AUTHOR", "EVENT", "IP ADDRESS", "DATE")
	fmt.Fprintf(tabWriter, header)

	// interactive mode is assumed, except when output is piped.
	interactive := !cmd.io.Stdout().IsPiped()

	iter := client.Secrets().EventIterator(cmd.path.Value(), nil)
	i := 0
	for {
		i++
		event, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err == api.ErrSecretNotFound {
			// Check if we're attempting to audit a dir.
			_, err = client.Dirs().GetTree(api.DirPath(cmd.path).Value(), 1, false)
			if err == nil {
				return ErrCannotAuditDir
			}
			return api.ErrSecretNotFound
		} else if err != nil {
			return err
		}

		actor, err := getAuditActor(event)
		if err != nil {
			return err
		}

		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n",
			actor,
			getEventAction(event),
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
