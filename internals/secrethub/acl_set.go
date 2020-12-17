package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	//"github.com/spf13/cobra"
)

// ACLSetCommand is a command to set access rules.
type ACLSetCommand struct {
	accountName api.AccountName
	force       bool
	io          ui.IO
	path        api.DirPath
	permission  api.Permission
	newClient   newClientFunc
}

// NewACLSetCommand creates a new ACLSetCommand.
func NewACLSetCommand(io ui.IO, newClient newClientFunc) *ACLSetCommand {
	return &ACLSetCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register adds a CommandClause and it's args and flags to a Registerer.
// Register adds args and flags.
func (cmd *ACLSetCommand) Register(r cli.Registerer) {
	clause := r.Command("set", "Set access rule for a user or service on a path.")
	registerForceFlag(clause, &cmd.force)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.Argument{
		{Store: &cmd.path, Name: "dir-path", Placeholder: dirPathPlaceHolder, Required: true, Description: "The path of the directory to set the access rule for."},
		{Store: &cmd.accountName, Name: "account-name", Required: true, Description: "The account name (username or service name) to set the access rule for."},
		{Store: &cmd.permission, Name: "permission", Required: true, Description: "The permission to set in the access rule."},
	})
}

// Run handles the command with the options as specified in the command.
func (cmd *ACLSetCommand) Run() error {
	if !cmd.force {
		confirmed, err := ui.AskYesNo(
			cmd.io,
			fmt.Sprintf(
				"[WARNING] This gives %s %s rights on all directories and secrets contained in %s. "+
					"Are you sure you want to set this access rule?",
				cmd.accountName,
				cmd.permission,
				cmd.path,
			),
			ui.DefaultNo,
		)
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}
	}

	fmt.Fprintf(cmd.io.Output(), "Setting access rule for %s at %s with %s\n", cmd.accountName, cmd.path, cmd.permission)

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	_, err = client.AccessRules().Set(cmd.path.Value(), cmd.permission.String(), cmd.accountName.Value())
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Access rule set!")

	return nil
}
