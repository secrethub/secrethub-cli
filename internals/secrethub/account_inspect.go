package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

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
func (cmd *AccountInspectCommand) Register(r Registerer) {
	clause := r.Command("inspect", "Show the details of your SecretHub account.")

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountInspectCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	user, err := client.Users().Me()
	if err != nil {
		return errio.Error(err)
	}

	output, err := cli.PrettyJSON(newOutputUser(user, cmd.timeFormatter))
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintln(cmd.io.Stdout(), output)

	return nil
}

// outputUser is a user friendly JSON representation of a user account.
type outputUser struct {
	Username         string
	FullName         string
	Email            string `json:",omitempty"`
	EmailVerified    bool   `json:",omitempty"`
	CreatedAt        string `json:",omitempty"`
	PublicAccountKey []byte `json:",omitempty"`
}

func newOutputUser(user *api.User, timeFormatter TimeFormatter) *outputUser {
	return &outputUser{
		Username:         user.Username,
		FullName:         user.FullName,
		Email:            user.Email,
		EmailVerified:    user.EmailVerified,
		CreatedAt:        timeFormatter.Format(user.CreatedAt.Local()),
		PublicAccountKey: user.PublicKey,
	}
}
