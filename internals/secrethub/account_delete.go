package secrethub

import (
	"fmt"
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
		return fmt.Errorf("cannot delete account that is a member of an organization. Please leave or delete the following organizations before deleting your account: %s", builder.String())
	}

	var confirmed bool
	confirmed, err = ui.ConfirmCaseInsensitive(
		cmd.io,
		fmt.Sprintf("[DANGER ZONE] This action cannot be undone. "+
			"This will permanently delete the account named %s along with all its repositories and secrets. "+
			"Please type in the name of the account to confirm", account.Name),
		account.Name.String(),
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.io.Output(), "Name does not match. Aborting.")
		return nil
	}

	err = client.Accounts().Delete(account.AccountID)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.io.Output(), "Your account was successfully deleted.")
	return nil
}
