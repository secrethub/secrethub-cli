package secrethub

import (
	"fmt"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// Errors
var (
	ErrLocalAccountFound = errMain.Code("local_account_found").Error("found a local account configuration. To overwrite it, run the same command with the --force or -f flag.")
)

// SignUpCommand signs up a new user and configures his account for use on this machine.
type SignUpCommand struct {
	username        string
	fullName        string
	email           string
	force           bool
	io              ui.IO
	newClient       newClientFunc
	credentialStore CredentialStore
	progressPrinter progress.Printer
}

// NewSignUpCommand creates a new SignUpCommand.
func NewSignUpCommand(io ui.IO, newClient newClientFunc, credentialStore CredentialStore) *SignUpCommand {
	return &SignUpCommand{
		io:              io,
		newClient:       newClient,
		credentialStore: credentialStore,
		progressPrinter: progress.NewPrinter(io.Stdout(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SignUpCommand) Register(r Registerer) {
	clause := r.Command("signup", "Signup for an account on SecretHub.")
	clause.Flag("username", "The username you would like to use on SecretHub. If not set, you will be asked for it.").StringVar(&cmd.username)
	clause.Flag("full-name", "If not set, you will be asked to provide your full name.").StringVar(&cmd.fullName)
	clause.Flag("email", "If not set, you will be asked to provide your email address.").StringVar(&cmd.email)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run signs up a new user and configures his account for use on this machine.
// If an account was already configured, the user is prompted for confirmation to overwrite it.
func (cmd *SignUpCommand) Run() error {
	profileDir, err := cmd.credentialStore.NewProfileDir()
	if err != nil {
		return errio.Error(err)
	}
	credentialPath := profileDir.CredentialPath()

	if cmd.force {
		if cmd.username == "" || cmd.fullName == "" || cmd.email == "" {
			return ErrMissingFlags
		}
	} else {
		exists, err := cmd.credentialStore.CredentialExists()
		if err != nil {
			return errio.Error(err)
		}
		if exists {
			confirmed, err := ui.AskYesNo(
				cmd.io,
				fmt.Sprintf("Found account credentials at %s, do you wish to overwrite them?", credentialPath),
				ui.DefaultNo,
			)
			if err == ui.ErrCannotAsk {
				return ErrLocalAccountFound
			} else if err != nil {
				return errio.Error(err)
			}

			if !confirmed {
				fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
				return nil
			}
		}

		if cmd.username == "" || cmd.fullName == "" || cmd.email == "" {
			_, promptOut, err := cmd.io.Prompts()
			if err != nil {
				return errio.Error(err)
			}
			fmt.Fprint(
				promptOut,
				"Let's get you setup. "+
					"Before we continue, I need to know a few things about you. "+
					"Please answer the questions below, followed by an [ENTER]\n\n",
			)
			if cmd.username == "" {
				cmd.username, err = ui.AskAndValidate(cmd.io, "The username you'd like to use: ", 2, api.ValidateUsername)
				if err != nil {
					return errio.Error(err)
				}
			}
			if cmd.fullName == "" {
				cmd.fullName, err = ui.AskAndValidate(cmd.io, "Your full name: ", 2, api.ValidateFullName)
				if err != nil {
					return errio.Error(err)
				}
			}
			if cmd.email == "" {
				cmd.email, err = ui.AskAndValidate(cmd.io, "Your email address: ", 2, api.ValidateEmail)
				if err != nil {
					return errio.Error(err)
				}
			}
		}
	}

	fmt.Fprintf(
		cmd.io.Stdout(),
		"An account credential will be generated and stored at %s. "+
			"Losing this credential means you lose the ability to decrypt your secrets. "+
			"So keep it safe.\n",
		credentialPath,
	)

	// Only prompt for a passphrase when the user hasn't used --force.
	// Otherwise, we assume the passphrase was intentionally not
	// configured to output a plaintext credential.
	if !cmd.credentialStore.IsPassphraseSet() && !cmd.force {
		passphrase, err := ui.AskPassphrase(cmd.io, "Please enter a passphrase to protect your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
		if err != nil {
			return errio.Error(err)
		}
		cmd.credentialStore.SetPassphrase(passphrase)
	}

	fmt.Fprint(cmd.io.Stdout(), "Generating credential...")
	cmd.progressPrinter.Start()
	credential, err := secrethub.GenerateCredential()
	if err != nil {
		return errio.Error(err)
	}
	cmd.credentialStore.Set(credential)
	cmd.progressPrinter.Stop()

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprint(cmd.io.Stdout(), "Signing you up...")
	cmd.progressPrinter.Start()
	_, err = client.Users().Create(cmd.username, cmd.email, cmd.fullName)
	cmd.progressPrinter.Stop()
	if err != nil {
		return errio.Error(err)
	}
	err = cmd.credentialStore.Save()
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintln(cmd.io.Stdout(), "Signup complete! You're now on SecretHub.")
	fmt.Fprintf(cmd.io.Stdout(), "We've send an email to %s Please verify your email address to continue.\n", cmd.email)
	return nil
}
