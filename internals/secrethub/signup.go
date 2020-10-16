package secrethub

import (
	"fmt"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
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
	username        string
	fullName        string
	email           string
	org             string
	orgDescription  string
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
		progressPrinter: progress.NewPrinter(io.Output(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SignUpCommand) Register(r cli.Registerer) {
	clause := r.Command("signup", "Create a free personal developer account.")
	clause.Flags().StringVar(&cmd.username, "username", "", "The username you would like to use on SecretHub.")
	clause.Flags().StringVar(&cmd.fullName, "full-name", "", "Your full name.")
	clause.Flags().StringVar(&cmd.email, "email", "", "Your (work) email address we will use for all correspondence.")
	clause.Flags().StringVar(&cmd.org, "org", "", "The name of your organization.")
	clause.Flags().StringVar(&cmd.orgDescription, "org-description", "", "A description (max 144 chars) for your organization so others will recognize it.")
	registerForceFlag(clause, &cmd.force)

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
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
				fmt.Fprintln(cmd.io.Output(), "Aborting.")
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
				cmd.email, err = ui.AskAndValidate(cmd.io, "Your (work) email address: ", 2, api.ValidateEmail)
				if err != nil {
					return err
				}
			}
			fmt.Fprintln(cmd.io.Output())
		}
	}

	fmt.Fprintf(cmd.io.Output(), credentialCreationMessage, credentialPath)

	// Only prompt for a passphrase when the user hasn't used --force.
	// Otherwise, we assume the passphrase was intentionally not
	// configured to output a plaintext credential.
	var passphrase string
	if !cmd.credentialStore.IsPassphraseSet() && !cmd.force {
		var err error
		passphrase, err = askCredentialPassphrase(cmd.io)
		if err != nil {
			return err
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.io.Output(), "Setting up your account...")
	cmd.progressPrinter.Start()
	credential := credentials.CreateKey()
	_, err = client.Users().Create(cmd.username, cmd.email, cmd.fullName, credential)
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	err = writeNewCredential(credential, passphrase, cmd.credentialStore.ConfigDir().Credential())
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	secretPath, err := createStartRepo(client, cmd.username, cmd.fullName)
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	cmd.progressPrinter.Stop()
	fmt.Fprint(cmd.io.Output(), "Created your account.\n\n")

	err = createWorkspace(client, cmd.io, cmd.org, cmd.orgDescription, cmd.progressPrinter)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Setup complete. To read your first secret, run:\n\n    secrethub read %s\n\n", secretPath)

	return nil
}

// createStartRepo creates a start repository and writes a fist secret to it, so that
// the user can start by reading their first secret. It returns the secret's path.
// This is intended to smoothen onboarding.
func createStartRepo(client secrethub.ClientInterface, username string, fullName string) (string, error) {
	repoPath := secretpath.Join(username, "start")
	_, err := client.Repos().Create(secretpath.Join(repoPath))
	if err != nil {
		return "", err
	}

	secretPath := secretpath.Join(repoPath, "hello")
	message := fmt.Sprintf("Welcome %s! This is your first secret. To write a new version of this secret, run:\n\n    secrethub write %s", fullName, secretPath)

	_, err = client.Secrets().Write(secretPath, []byte(message))
	if err != nil {
		return "", err
	}
	return secretPath, nil
}

// createWorkspace creates a new org with the given name and description.
func createWorkspace(client secrethub.ClientInterface, io ui.IO, org string, orgDescription string, progressPrinter progress.Printer) error {
	if org == "" {
		createWorkspace, err := ui.AskYesNo(io, "Do you want to create a shared workspace for your team?", ui.DefaultYes)
		if err != nil {
			return err
		}
		fmt.Fprintln(io.Output())
		if !createWorkspace {
			fmt.Fprint(io.Output(), "You can create a shared workspace later using `secrethub org init`.\n\n")
			return nil
		}
	}

	var err error
	if org == "" {
		org, err = ui.AskAndValidate(io, "Workspace name (e.g. your company name): ", 2, api.ValidateOrgName)
		if err != nil {
			return err
		}
	}
	if orgDescription == "" {
		orgDescription, err = ui.AskAndValidate(io, "A description (max 144 chars) for your team workspace so others will recognize it:\n", 2, api.ValidateOrgDescription)
		if err != nil {
			return err
		}
	}

	fmt.Fprint(io.Output(), "Creating your shared workspace...")
	progressPrinter.Start()

	_, err = client.Orgs().Create(org, orgDescription)
	progressPrinter.Stop()
	if err == api.ErrOrgAlreadyExists {
		fmt.Fprintf(io.Output(), "The workspace %s already exists. If it is your organization, ask a colleague to invite you to the workspace. You can also create a new one using `secrethub org init`.\n", org)
	} else if err != nil {
		return err
	} else {
		fmt.Fprint(io.Output(), "Created your shared workspace.\n\n")
	}
	return nil
}

// writeCredential writes the given credential to the configuration directory.
func writeNewCredential(credential *credentials.KeyCreator, passphrase string, credentialFile *configdir.CredentialFile) error {
	exportKey := credential.Key
	if passphrase != "" {
		exportKey = exportKey.Passphrase(credentials.FromString(passphrase))
	}

	encodedCredential, err := credential.Export()
	if err != nil {
		return err
	}

	return credentialFile.Write(encodedCredential)
}

// askCredentialPassphrase prompts the user for a passphrase to protect the local credential.
func askCredentialPassphrase(io ui.IO) (string, error) {
	return ui.AskPassphrase(io, "Please enter a passphrase to protect your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
}
