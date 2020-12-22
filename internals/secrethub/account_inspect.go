package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

const accountTypeUser string = "user"
const accountTypeService string = "service"

// AccountInspectCommand is a command to inspect account details.
type AccountInspectCommand struct {
	io            ui.IO
	newClient     newClientFunc
	timeFormatter TimeFormatter
}

// NewAccountInspectCommand creates a new AccountInspectCommand.
func NewAccountInspectCommand(io ui.IO, newClient newClientFunc) *AccountInspectCommand {
	return &AccountInspectCommand{
		io:            io,
		newClient:     newClient,
		timeFormatter: NewTimeFormatter(true),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AccountInspectCommand) Register(r command.Registerer) {
	clause := r.Command("inspect", "Show the details of your SecretHub account.")

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountInspectCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	account, err := client.Accounts().Me()
	if err != nil {
		return err
	}
	var output string
	if account.AccountType == accountTypeUser {
		user, err := client.Users().Me()
		if err != nil {
			return err
		}
		output, err = cli.PrettyJSON(newOutputUser(user, cmd.timeFormatter))
		if err != nil {
			return err
		}
	} else if account.AccountType == accountTypeService {
		service, err := client.Services().Get(account.Name.String())
		if err != nil {
			return err
		}
		output, err = cli.PrettyJSON(newOutputService(service, account, cmd.timeFormatter))
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(cmd.io.Output(), output)

	return nil
}

// outputAccount contains the fields common in both outputUser and outputService
type outputAccount struct {
	AccountType      string
	AccountName      string
	CreatedAt        string `json:",omitempty"`
	PublicAccountKey []byte `json:",omitempty"`
}

// outputUser is a user friendly JSON representation of a user account.
type outputUser struct {
	Username      string
	FullName      string
	Email         string `json:",omitempty"`
	EmailVerified bool   `json:",omitempty"`
	outputAccount
}

func newOutputUser(user *api.User, timeFormatter TimeFormatter) *outputUser {
	return &outputUser{
		Username:      user.Username,
		FullName:      user.FullName,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		outputAccount: outputAccount{
			AccountType:      accountTypeUser,
			AccountName:      user.Username,
			CreatedAt:        timeFormatter.Format(user.CreatedAt.Local()),
			PublicAccountKey: user.PublicKey,
		},
	}
}

// outputService is a user friendly JSON representation of a service account.
type outputService struct {
	Description string
	CreatedAt   string `json:",omitempty"`
	outputAccount
}

func newOutputService(service *api.Service, account *api.Account, timeFormatter TimeFormatter) *outputService {
	return &outputService{
		Description: service.Description,
		outputAccount: outputAccount{
			AccountType:      accountTypeService,
			AccountName:      service.ServiceID,
			CreatedAt:        timeFormatter.Format(service.CreatedAt.Local()),
			PublicAccountKey: account.PublicKey,
		},
	}
}
