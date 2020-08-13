package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

// OrgLsCommand handles listing all organisations a user is a member of.
type OrgLsCommand struct {
	quiet         bool
	useTimestamps bool
	io            ui.IO
	newClient     newClientFunc
	timeFormatter TimeFormatter
}

// NewOrgLsCommand creates a new OrgLsCommand.
func NewOrgLsCommand(io ui.IO, newClient newClientFunc) *OrgLsCommand {
	return &OrgLsCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgLsCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("ls", "List all organizations you are a member of.")
	clause.Alias("list")
	clause.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false, "Only print organization names.")

	registerTimestampFlag(clause, &cmd.useTimestamps)

	command.BindAction(clause, nil, cmd.Run)
}

// Run lists all organizations a user is a member of.
func (cmd *OrgLsCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *OrgLsCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// Run lists all organizations a user is a member of.
func (cmd *OrgLsCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	resp, err := client.Orgs().ListMine()
	if err != nil {
		return err
	}

	sort.Sort(api.SortOrgByName(resp))

	if cmd.quiet {
		for _, org := range resp {
			fmt.Fprintf(cmd.io.Output(), "%s\n", org.Name)
		}
	} else {
		w := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "NAME", "REPOS", "USERS", "CREATED")

		for _, org := range resp {
			// TODO SHDEV-724: refactor these two calls to include the counts in the api.Org response by default.
			members, err := client.Orgs().Members().List(org.Name)
			if err != nil {
				return err
			}

			repos, err := client.Repos().List(org.Name)
			if err != nil {
				return err
			}

			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", org.Name, len(repos), len(members), cmd.timeFormatter.Format(org.CreatedAt.Local()))
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}
