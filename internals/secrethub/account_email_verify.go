package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-cli/internals/cli"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
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
func (cmd *AccountEmailVerifyCommand) Register(r cli.Registerer) {
	clause := r.Command("verify-email", "Resend verification email to the registered email address.")
	clause.HelpLong("When you create your account, a verification email is automatically sent to the email address you used to sign up. " +
		"In case anything goes wrong (e.g. the email ended up in your junk folder), this command lets you resend the verification email. " +
		"Once received, click the link in the verification email to verify your email address.")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountEmailVerifyCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	user, err := client.Me().GetUser()
	if err != nil {
		return err
	}

	if user.EmailVerified {
		fmt.Fprintln(cmd.io.Output(), "Your email address is already verified.")
		return nil
	}

	err = client.Me().SendVerificationEmail()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "An email has been sent to %s with an email verification link. Please check your mail and click the link.\n\n", user.Email)

	fmt.Fprintf(cmd.io.Output(), "Please contact support@secrethub.io if the problem persists.\n\n")

	return nil
}
