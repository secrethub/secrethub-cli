package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-go/pkg/secretpath"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

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
func (cmd *ACLCheckCommand) Register(r command.Registerer) {
	clause := r.Command("check", "Checks the effective permission of accounts on a path.")
	clause.Arg("dir-path", "The path of the directory to check the effective permission for").Required().PlaceHolder(optionalDirPathPlaceHolder).SetValue(&cmd.path)
	clause.Arg("account-name", "Check permissions of a specific account name (username or service name). When left empty, all accounts with permission on the path are printed out.").SetValue(&cmd.accountName)

	command.BindAction(clause, cmd.Run)
}

// Run prints the access level(s) on the given directory.
func (cmd *ACLCheckCommand) Run() error {
	levels, err := cmd.listLevels()
	if err != nil {
		return err
	}

	if cmd.accountName != "" {
		for _, level := range levels {
			if level.Account.Name == cmd.accountName {
				fmt.Fprintf(cmd.io.Stdout(), "%s\n", level.Permission.String())
				return nil
			}
		}

		fmt.Fprintln(cmd.io.Stdout(), api.PermissionNone.String())
		return nil
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

func (cmd *ACLCheckCommand) listLevels() ([]*api.AccessLevel, error) {
	client, err := cmd.newClient()
	if err != nil {
		return nil, err
	}

	path := cmd.path.Value()

	levels, err := client.AccessRules().ListLevels(path)
	if err == nil {
		return levels, nil
	}
	if !api.IsErrNotFound(err) {
		return nil, err
	}

	isSecret, isSecretErr := client.Secrets().Exists(path)
	if isSecretErr != nil {
		return nil, err
	}
	if isSecret {
		levels, err = client.AccessRules().ListLevels(secretpath.Parent(path))
		if err != nil {
			return nil, err
		}
		return levels, nil
	}
	return nil, err

}
