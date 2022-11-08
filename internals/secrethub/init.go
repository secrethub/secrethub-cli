package secrethub

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/configdir"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
)

const signupMessage = "Go to https://signup.secrethub.io/ and follow the steps to create an account and get it set up on this machine."

// InitCommand configures the user's SecretHub account for use on this machine.
type InitCommand struct {
	backupCode               string
	setupCode                string
	force                    bool
	io                       ui.IO
	newClientWithCredentials func(credentials.Provider) (secrethub.ClientInterface, error)
	credentialStore          CredentialConfig
	progressPrinter          progress.Printer
}

// NewInitCommand creates a new InitCommand.
func NewInitCommand(io ui.IO, newClientWithCredentials func(credentials.Provider) (secrethub.ClientInterface, error), credentialStore CredentialConfig) *InitCommand {
	return &InitCommand{
		io:                       io,
		newClientWithCredentials: newClientWithCredentials,
		credentialStore:          credentialStore,
		progressPrinter:          progress.NewPrinter(io.Output(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *InitCommand) Register(r cli.Registerer) {
	clause := r.Command("init", "Initialize the SecretHub client for first use on this device.")
	clause.Flags().StringVar(&cmd.backupCode, "backup-code", "", "The backup code used to restore an existing account to this device.")
	clause.Flags().StringVar(&cmd.setupCode, "setup-code", "", "The setup code used to configure the CLI to use an account created on the website.")
	registerForceFlag(clause, &cmd.force)

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

type InitMode int

const (
	InitModeBackupCode InitMode = iota + 1
	InitModeSetupCode
)

// Run configures the user's SecretHub account for use on this machine.
// If an account was already configured, the user is prompted for confirmation to overwrite it.
func (cmd *InitCommand) Run() error {
	if cmd.setupCode != "" && cmd.backupCode != "" {
		return ErrFlagsConflict("--backup-code and --setup-code")
	}

	credentialPath := cmd.credentialStore.ConfigDir().Credential().Path()

	if cmd.credentialStore.ConfigDir().Credential().Exists() && !cmd.force {
		confirmed, err := ui.AskYesNo(
			cmd.io,
			fmt.Sprintf("Already found a credential at %s, do you wish the re-initialize SecretHub on this device? (this will overwrite the credential)", credentialPath),
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

	var mode InitMode
	if cmd.setupCode != "" {
		mode = InitModeSetupCode
	} else if cmd.backupCode != "" {
		mode = InitModeBackupCode
	}

	if mode == 0 {
		if cmd.force {
			return ErrMissingFlags
		}
		option, err := ui.Choose(cmd.io, "How do you want to initialize your SecretHub account on this device?",
			[]string{
				"Sign up for a new account",
				"Use a backup code to recover an existing account",
			}, 3)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.io.Output())

		switch option {
		case 0:
			fmt.Fprintln(cmd.io.Output(), signupMessage)
			return nil
		case 1:
			mode = InitModeBackupCode
		}
	}

	switch mode {
	case InitModeSetupCode:
		setupCode := cmd.setupCode

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

		deviceName, err := promptForDeviceName(cmd.io)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.io.Output(), "Setting up your account...")
		cmd.progressPrinter.Start()

		client, err := cmd.newClientWithCredentials(credentials.NewSetupCode(setupCode))
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}

		credential := credentials.CreateKey()
		_, err = client.Credentials().Create(credential, deviceName)
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}

		err = writeNewCredential(credential, passphrase, cmd.credentialStore.ConfigDir().Credential())
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}

		client, err = cmd.newClientWithCredentials(credential)
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}

		me, err := client.Me().GetUser()
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}

		secretPath, err := createStartRepo(client, me.Username, me.FullName)
		if err != nil {
			cmd.progressPrinter.Stop()
			return err
		}
		cmd.progressPrinter.Stop()
		fmt.Fprint(cmd.io.Output(), "Created your account.\n\n")

		err = createWorkspace(client, cmd.io, "", "", cmd.progressPrinter)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "Setup complete. To read your first secret, run:\n\n    secrethub read %s\n\n", secretPath)
		return nil
	case InitModeBackupCode:
		backupCode := cmd.backupCode

		if backupCode == "" {
			var err error
			backupCode, err = ui.AskAndValidate(cmd.io, "What is your backup code?\n", 3, credentials.ValidateBootstrapCode)
			if err != nil {
				return err
			}
		}

		client, err := cmd.newClientWithCredentials(credentials.UseBackupCode(backupCode))
		if err != nil {
			return err
		}

		me, err := client.Me().GetUser()
		if err != nil {
			statusErr, ok := err.(errio.PublicStatusError)
			if ok && statusErr.Code == "invalid_signature" {
				return errors.New("this backup code is not found on the server")
			}
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "This backup code can be used to recover the account `%s`\n", me.Username)
		ok, err := ui.AskYesNo(cmd.io, "Do you want to continue?", ui.DefaultYes)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}

		deviceName, err := promptForDeviceName(cmd.io)
		if err != nil {
			return err
		}

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

		credential := credentials.CreateKey()
		_, err = client.Credentials().Create(credential, deviceName)
		if err != nil {
			return err
		}

		exportKey := credential.Key
		if passphrase != "" {
			exportKey = exportKey.Passphrase(credentials.FromString(passphrase))
		}

		exportedKey, err := exportKey.Export()
		if err != nil {
			return err
		}
		err = cmd.credentialStore.ConfigDir().Credential().Write(exportedKey)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid option")
	}
}

func promptForDeviceName(io ui.IO) (string, error) {
	deviceName := ""
	question := "What is the name of this device?"
	hostName, err := os.Hostname()
	if err == nil {
		deviceName, err = ui.AskWithDefault(io, question, hostName)
		if err != nil {
			return "", err
		}
	} else {
		deviceName, err = ui.Ask(io, question)
		if err != nil {
			return "", err
		}
	}
	return deviceName, nil
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

	encodedCredential, err := exportKey.Export()
	if err != nil {
		return err
	}

	return credentialFile.Write(encodedCredential)
}

// askCredentialPassphrase prompts the user for a passphrase to protect the local credential.
func askCredentialPassphrase(io ui.IO) (string, error) {
	return ui.AskPassphrase(io, "Please enter a passphrase to protect your local credential (leave empty for no passphrase): ", "Enter the same passphrase again: ", 3)
}
