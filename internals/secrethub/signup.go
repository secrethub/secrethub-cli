package secrethub

import (
	"fmt"
	"io"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
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
	credentialStore CredentialConfig
	progressPrinter progress.Printer
}

// NewSignUpCommand creates a new SignUpCommand.
func NewSignUpCommand(io ui.IO, newClient newClientFunc, credentialStore CredentialConfig) *SignUpCommand {
	return &SignUpCommand{
		io:              io,
		newClient:       newClient,
		credentialStore: credentialStore,
		progressPrinter: progress.NewPrinter(io.Stdout(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SignUpCommand) Register(r Registerer) {
	clause := r.Command("signup", "Create a free personal developer account.")
	clause.Flag("username", "The username you would like to use on SecretHub. If not set, you will be asked for it.").StringVar(&cmd.username)
	clause.Flag("full-name", "If not set, you will be asked to provide your full name.").StringVar(&cmd.fullName)
	clause.Flag("email", "If not set, you will be asked to provide your email address.").StringVar(&cmd.email)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run signs up a new user and configures his account for use on this machine.
// If an account was already configured, the user is prompted for confirmation to overwrite it.
func (cmd *SignUpCommand) Run() error {
	credentialPath := cmd.credentialStore.ConfigDir().Credential().Path()

	if cmd.force {
		if cmd.username == "" || cmd.fullName == "" || cmd.email == "" {
			return ErrMissingFlags
		}
	} else {
		if cmd.credentialStore.ConfigDir().Credential().Exists() {
			confirmed, err := ui.AskYesNo(
				cmd.io,
				fmt.Sprintf("Found account credentials at %s, do you wish to overwrite them?", credentialPath),
				ui.DefaultNo,
			)
			if err == ui.ErrCannotAsk {
				return ErrLocalAccountFound
			} else if err != nil {
				return err
			}

			if !confirmed {
				fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
				return nil
			}
		}

		if cmd.username == "" || cmd.fullName == "" || cmd.email == "" {
			_, promptOut, err := cmd.io.Prompts()
			if err != nil {
				return err
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
					return err
				}
			}
			if cmd.fullName == "" {
				cmd.fullName, err = ui.AskAndValidate(cmd.io, "Your full name: ", 2, api.ValidateFullName)
				if err != nil {
					return err
				}
			}
			if cmd.email == "" {
				cmd.email, err = ui.AskAndValidate(cmd.io, "Your email address: ", 2, api.ValidateEmail)
				if err != nil {
					return err
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
	var passphrase string
	if !cmd.credentialStore.IsPassphraseSet() && !cmd.force {
		var err error
		passphrase, err = ui.AskPassphrase(cmd.io, "Please enter a passphrase to protect your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
		if err != nil {
			return err
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.io.Stdout(), "Signing you up...")
	cmd.progressPrinter.Start()
	credential := credentials.CreateKey()
	_, err = client.Users().Create(cmd.username, cmd.email, cmd.fullName, credential)
	cmd.progressPrinter.Stop()
	if err != nil {
		return err
	}

	exportKey := credential.Key
	if passphrase != "" {
		exportKey = exportKey.Passphrase(credentials.FromString(passphrase))
	}

	encodedCredential, err := credential.Export()
	if err != nil {
		return err
	}
	err = cmd.credentialStore.ConfigDir().Credential().Write(encodedCredential)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Stdout(), "Signup complete! You're now on SecretHub.")

	return createStartRepo(client, cmd.io.Stdout(), cmd.username, cmd.fullName)
}

// createStartRepo creates a start repository and write a fist secret to it, so that
// the user can start by reading their first secret. This is intended to smoothen
// onboarding.
func createStartRepo(client secrethub.ClientInterface, w io.Writer, workspace string, name string) error {
	fmt.Fprintln(w, "Setting up your workspace...")
	repoPath := secretpath.Join(workspace, "start")
	_, err := client.Repos().Create(secretpath.Join(repoPath))
	if err != nil {
		return err
	}

	secretPath := secretpath.Join(repoPath, "hello")
	message := fmt.Sprintf("Welcome %s! This is your first secret. To write a new version of this secret, run:\n\n    secrethub write %s", name, secretPath)

	_, err = client.Secrets().Write(secretPath, []byte(message))
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Setup complete. To read your first secret, run:\n\n    secrethub read %s\n\n", secretPath)
	return nil
}
