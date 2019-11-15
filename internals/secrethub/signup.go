package secrethub

import (
	"fmt"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
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
		progressPrinter: progress.NewPrinter(io.Stdout(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *SignUpCommand) Register(r command.Registerer) {
	clause := r.Command("signup", "Create a free personal developer account.")
	clause.Flag("username", "The username you would like to use on SecretHub.").StringVar(&cmd.username)
	clause.Flag("full-name", "Your full name.").StringVar(&cmd.fullName)
	clause.Flag("email", "Your (work) email address we will use for all correspondence.").StringVar(&cmd.email)
	clause.Flag("org", "The name of your organization.").StringVar(&cmd.org)
	clause.Flag("org-description", "A description (max 144 chars) for your organization so others will recognize it.").StringVar(&cmd.orgDescription)
	registerForceFlag(clause).BoolVar(&cmd.force)

	command.BindAction(clause, cmd.Run)
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
				cmd.email, err = ui.AskAndValidate(cmd.io, "Your (work) email address: ", 2, api.ValidateEmail)
				if err != nil {
					return err
				}
			}
			fmt.Fprintln(cmd.io.Stdout())
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

	fmt.Fprint(cmd.io.Stdout(), "Setting up your account...")
	cmd.progressPrinter.Start()
	credential := credentials.CreateKey()
	_, err = client.Users().Create(cmd.username, cmd.email, cmd.fullName, credential)
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	exportKey := credential.Key
	if passphrase != "" {
		exportKey = exportKey.Passphrase(credentials.FromString(passphrase))
	}

	encodedCredential, err := credential.Export()
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}
	err = cmd.credentialStore.ConfigDir().Credential().Write(encodedCredential)
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	// create a start repository and write a fist secret to it, so that
	// the user can start by reading their first secret.
	// This is intended to smoothen onboarding.
	repoPath := secretpath.Join(cmd.username, "start")
	_, err = client.Repos().Create(secretpath.Join(repoPath))
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	secretPath := secretpath.Join(repoPath, "hello")
	message := fmt.Sprintf("Welcome %s! This is your first secret. To write a new version of this secret, run:\n\n    secrethub write %s", cmd.fullName, secretPath)

	_, err = client.Secrets().Write(secretPath, []byte(message))
	if err != nil {
		cmd.progressPrinter.Stop()
		return err
	}

	cmd.progressPrinter.Stop()
	fmt.Fprint(cmd.io.Stdout(), "Created your account.\n\n")

	createWorkspace := cmd.org != ""
	if !createWorkspace {
		createWorkspace, err = ui.AskYesNo(cmd.io, "Do you want to create a shared workspace for your team?", ui.DefaultYes)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.io.Stdout())
		if !createWorkspace {
			fmt.Fprint(cmd.io.Stdout(), "You can create a shared workspace later using `secrethub org init`.\n\n")
		}
	}
	if createWorkspace {
		if cmd.org == "" {
			cmd.org, err = ui.AskAndValidate(cmd.io, "Workspace name (e.g. your company name): ", 2, api.ValidateOrgName)
			if err != nil {
				return err
			}
		}
		if cmd.orgDescription == "" {
			cmd.orgDescription, err = ui.AskAndValidate(cmd.io, "A description (max 144 chars) for your team workspace so others will recognize it:\n", 2, api.ValidateOrgDescription)
			if err != nil {
				return err
			}
		}

		_, err := client.Orgs().Create(cmd.org, cmd.orgDescription)
		if err == api.ErrOrgAlreadyExists {
			fmt.Fprintf(cmd.io.Stdout(), "The workspace %s already exists. If it is your organization, ask a colleague to invite you to the workspace. You can also create a new one using `secrethub org init`.\n", cmd.org)
		} else if err != nil {
			return err
		} else {
			fmt.Fprintln(cmd.io.Stdout(), "\nCreated your shared workspace.")
		}
	}
	fmt.Fprintf(cmd.io.Stdout(), "Setup complete. To read your first secret, run:\n\n    secrethub read %s\n\n", secretPath)

	return nil
}
