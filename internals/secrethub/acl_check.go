package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
)

// ACLCheckCommand prints the access level(s) on a given directory.
type ACLCheckCommand struct {
	path        api.DirPath
	accountName api.AccountName
	io          ui.IO
	newClient   newClientFunc
}

// NewACLCheckCommand creates a new ACLCheckCommand.
func NewACLCheckCommand(io ui.IO, newClient newClientFunc) *ACLCheckCommand {
	return &ACLCheckCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ACLCheckCommand) Register(r Registerer) {
	clause := r.Command("check", "Checks the effective permission of accounts on a path.")
	clause.Arg("dir-path", "The path of the directory to check the effective permission for (<namespace>/<repo>[/<dir>])").Required().SetValue(&cmd.path)
	clause.Arg("account-name", "Check permissions of a specific account name (username or service name). When left empty, all accounts with permission on the path are printed out.").SetValue(&cmd.accountName)

	BindAction(clause, cmd.Run)
}

// Run prints the access level(s) on the given directory.
func (cmd *ACLCheckCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	if cmd.accountName != "" {
		level, err := client.AccessRules().Get(cmd.path.Value(), cmd.accountName.Value())
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Stdout(), "%s\n", level.Permission.String())
		return nil
	}

	levels, err := client.AccessRules().ListLevels(cmd.path.Value())
	if err != nil {
		return err
	}

	sort.Sort(api.SortAccessLevels(levels))

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\n", "PERMISSIONS", "ACCOUNT")

	for _, level := range levels {
		fmt.Fprintf(tabWriter, "%s\t%s\n",
			level.Permission,
			level.Account.Name,
		)
	}

	err = tabWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}
