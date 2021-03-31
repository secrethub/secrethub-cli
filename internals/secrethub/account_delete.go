package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
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
	return client.Accounts().Delete(account.AccountID)
}
