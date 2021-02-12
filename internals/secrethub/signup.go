package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// Errors
var (
	ErrLocalAccountFound = errMain.Code("local_account_found").Error("found a local account configuration. To overwrite it, run the same command with the --force or -f flag.")
)

const credentialCreationMessage = "An account credential will be generated and stored at %s. " +
	"Losing this credential means you lose the ability to decrypt your secrets. " +
	"So keep it safe.\n"

// SignUpCommand signs up a new user and configures his account for use on this machine.
type SignUpCommand struct {
	io ui.IO
}

// NewSignUpCommand creates a new SignUpCommand.
func NewSignUpCommand(io ui.IO) *SignUpCommand {
	return &SignUpCommand{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SignUpCommand) Register(r cli.Registerer) {
	clause := r.Command("signup", "Create a free personal developer account.").Hidden()
	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run signs up a new user and configures his account for use on this machine.
// If an account was already configured, the user is prompted for confirmation to overwrite it.
func (cmd *SignUpCommand) Run() error {
	fmt.Fprintln(cmd.io.Output(), signupMessage)
	return nil
}
