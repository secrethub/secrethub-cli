package secrethub

import (
	"errors"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"
)

// AccountDeleteCommand is a command to inspect account details.
type AccountDeleteCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewAccountDeleteCommand creates a new AccountDeleteCommand.
func NewAccountDeleteCommand(io ui.IO, newClient newClientFunc) *AccountDeleteCommand {
	return &AccountDeleteCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AccountDeleteCommand) Register(r cli.Registerer) {
	clause := r.Command("delete", "Delete your SecretHub account.")

	clause.BindAction(cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountDeleteCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}
	var account *api.Account
	account, err = client.Accounts().Me()
	if err != nil {
		return err
	}

	var orgList []api.Org
	orgIterator := client.Orgs().Iterator(nil)
	for {
		var org api.Org
		org, err = orgIterator.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}
		orgList = append(orgList, org)
	}
	if len(orgList) > 0 {
		builder := strings.Builder{}
		for i, org := range orgList {
			builder.WriteString(org.Name)
			if i != len(orgList)-1 {
				builder.WriteString(", ")
			}
		}
		return errors.New("cannot delete account that is a member of an organization. Please leave or delete the following organizations before deleting your account: " + builder.String() + ".")
	}

	return client.Accounts().Delete(account.AccountID)
}
