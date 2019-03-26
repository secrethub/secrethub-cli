package secrethub

import (
	"fmt"

	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// AccountEmailVerifyCommand is a command to inspect account details.
type AccountEmailVerifyCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewAccountEmailVerifyCommand creates a new AccountEmailVerifyCommand.
func NewAccountEmailVerifyCommand(io ui.IO, newClient newClientFunc) *AccountEmailVerifyCommand {
	return &AccountEmailVerifyCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AccountEmailVerifyCommand) Register(r Registerer) {
	clause := r.Command("verify-email", "Send an email to the registered email address to prove you own that email address.")

	BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountEmailVerifyCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	user, err := client.Me().GetUser()
	if err != nil {
		return errio.Error(err)
	}

	if user.EmailVerified {
		fmt.Fprintln(cmd.io.Stdout(), "Your email address is already verified.")
		return nil
	}

	err = client.Me().SendVerificationEmail()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "An email has been sent to %s with an email verification link. Please check your mail and click the link.\n", user.Email)

	return nil
}
