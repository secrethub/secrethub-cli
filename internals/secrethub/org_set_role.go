package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	// "github.com/spf13/cobra"
)

// OrgSetRoleCommand handles updating the role of an organization member.
type OrgSetRoleCommand struct {
	orgName   api.OrgName
	username  cli.StringArgValue
	role      cli.StringArgValue
	io        ui.IO
	newClient newClientFunc
}

// NewOrgSetRoleCommand creates a new OrgSetRoleCommand.
func NewOrgSetRoleCommand(io ui.IO, newClient newClientFunc) *OrgSetRoleCommand {
	return &OrgSetRoleCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgSetRoleCommand) Register(r cli.Registerer) {
	clause := r.Command("set-role", "Set a user's organization role.")
	// clause.Cmd.Args = cobra.ExactValidArgs(3)
	//clause.Arg("org-name", "The organization name").Required().SetValue(&cmd.orgName)
	//clause.Arg("username", "The username of the user").Required().StringVar(&cmd.username)
	//clause.Arg("role", "The role to assign to the user. Can be either `admin` or `member`.").Required().StringVar(&cmd.role)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.orgName, &cmd.username, &cmd.role}, []string{"org-name", "username", "role"})
}

// Run updates the role of an organization member.
func (cmd *OrgSetRoleCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Setting role...\n")

	resp, err := client.Orgs().Members().Update(cmd.orgName.Value(), cmd.username.Param, cmd.role.Param)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Set complete! The user %s is %s of the %s organization.\n", resp.User.Username, resp.Role, cmd.orgName)

	return nil
}
