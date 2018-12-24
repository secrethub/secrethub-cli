package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
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
func (cmd *OrgLsCommand) Register(r Registerer) {
	clause := r.Command("ls", "List all organizations you are a member of.")
	clause.Flag("quiet", "Only print organization names.").Short('q').BoolVar(&cmd.quiet)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	BindAction(clause, cmd.Run)
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
		return errio.Error(err)
	}

	resp, err := client.Orgs().ListMine()
	if err != nil {
		return errio.Error(err)
	}

	sort.Sort(api.SortOrgByName(resp))

	if cmd.quiet {
		for _, org := range resp {
			fmt.Fprintf(cmd.io.Stdout(), "%s\n", org.Name)
		}
	} else {
		w := tabwriter.NewWriter(cmd.io.Stdout(), 0, 2, 2, ' ', 0)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "NAME", "REPOS", "USERS", "CREATED")

		for _, org := range resp {
			// TODO SHDEV-724: refactor these two calls to include the counts in the api.Org response by default.
			members, err := client.Orgs().Members().List(org.Name)
			if err != nil {
				return errio.Error(err)
			}

			repos, err := client.Repos().List(org.Name)
			if err != nil {
				return errio.Error(err)
			}

			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", org.Name, len(repos), len(members), cmd.timeFormatter.Format(org.CreatedAt.Local()))
		}

		err = w.Flush()
		if err != nil {
			return errio.Error(err)
		}
	}

	return nil
}
