package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// CredentialBackupCommand creates a backup code to restore a credential from a code.
type CredentialBackupCommand struct {
	io        ui.IO
	newClient newClientFunc
}

// NewAccountInitCommand creates a new AccountInitCommand.
func NewCredentialBackupCommand(io ui.IO, newClient newClientFunc) *CredentialBackupCommand {
	return &CredentialBackupCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CredentialBackupCommand) Register(r cli.Registerer) {
	clause := r.Command("backup", "Create a backup code for restoring your account.")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run creates a backup code for the currently authenticated account.
func (cmd *CredentialBackupCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	// Get username and make sure client has a valid credential.
	me, err := client.Me().GetUser()
	if err != nil {
		return err
	}

	question := fmt.Sprintf("This will create a new backup code for %s. "+
		"This code can be used to obtain full access to your account.\n"+
		"Do you want to continue?", me.Username)
	ok, err := ui.AskYesNo(cmd.io, question, ui.DefaultYes)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(cmd.io.Output(), "Aborting")
		return nil
	}

	backupCode := credentials.CreateBackupCode()

	_, err = client.Credentials().Create(backupCode, "")
	if err != nil {
		return err
	}

	code, err := backupCode.Code()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "This is your backup code: \n%s\n", code)
	fmt.Fprintln(cmd.io.Output(), "Write it down and store it in a safe location! "+
		"You can restore your account by running `secrethub init`.")

	return nil
}
