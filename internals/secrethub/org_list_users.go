package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
)

// OrgListUsersCommand handles listing the users of an organization.
type OrgListUsersCommand struct {
	orgName       api.OrgName
	useTimestamps bool
	io            ui.IO
	newClient     newClientFunc
	timeFormatter TimeFormatter
}

// NewOrgListUsersCommand creates a new OrgListUsersCommand.
func NewOrgListUsersCommand(io ui.IO, newClient newClientFunc) *OrgListUsersCommand {
	return &OrgListUsersCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgListUsersCommand) Register(r Registerer) {
	clause := r.Command("list-users", "List all members of an organization.")
	clause.Arg("org-name", "The organization name").Required().SetValue(&cmd.orgName)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	BindAction(clause, cmd.Run)
}

// Run lists the users of an organization.
func (cmd *OrgListUsersCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *OrgListUsersCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// run lists the users of an organization.
func (cmd *OrgListUsersCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	resp, err := client.Orgs().Members().List(cmd.orgName.Value())
	if err != nil {
		return err
	}

	sort.Sort(api.SortOrgMemberByUsername(resp))

	w := tabwriter.NewWriter(cmd.io.Stdout(), 0, 2, 2, ' ', 0)

	fmt.Fprintf(w, "%s\t%s\t%s\n", "USER", "ROLE", "LAST CHANGED")
	for _, member := range resp {
		fmt.Fprintf(w, "%s\t%s\t%s\n", member.User.Username, member.Role, cmd.timeFormatter.Format(member.LastChangedAt.Local()))
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}
