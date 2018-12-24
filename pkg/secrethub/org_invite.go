package secrethub

import (
	"fmt"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// OrgInviteCommand handles inviting a user to an organization.
type OrgInviteCommand struct {
	orgName   api.OrgName
	username  string
	role      string
	force     bool
	io        ui.IO
	newClient newClientFunc
}

// NewOrgInviteCommand creates a new OrgInviteCommand.
func NewOrgInviteCommand(io ui.IO, newClient newClientFunc) *OrgInviteCommand {
	return &OrgInviteCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgInviteCommand) Register(r Registerer) {
	clause := r.Command("invite", "Invite a user to join an organization.")
	clause.Arg("org-name", "The organization name").Required().SetValue(&cmd.orgName)
	clause.Arg("username", "The username of the user to invite").Required().StringVar(&cmd.username)
	clause.Flag("role", "Assign a role to the invited member. This can be either `admin` or `member`. It defaults to `member`.").Default("member").StringVar(&cmd.role)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run invites a user to an organization and gives them a certain role.
func (cmd *OrgInviteCommand) Run() error {
	if !cmd.force {
		msg := fmt.Sprintf("Are you sure you want to invite %s to the %s organization?",
			cmd.username,
			cmd.orgName)

		confirmed, err := ui.AskYesNo(cmd.io, msg, ui.DefaultNo)
		if err != nil {
			return errio.Error(err)
		}

		if !confirmed {
			fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
			return nil
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintln(cmd.io.Stdout(), "Inviting user...")

	resp, err := client.Orgs().Members().Invite(cmd.orgName.Value(), cmd.username, cmd.role)
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintf(cmd.io.Stdout(), "Invite complete! The user %s is now %s of the %s organization.\n", resp.User.Username, resp.Role, cmd.orgName)

	return nil
}
