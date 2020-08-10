package secrethub

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// InitCommand configures the user's SecretHub account for use on this machine.
type InitCommand struct {
	backupCode                  string
	setupCode                   string
	force                       bool
	io                          ui.IO
	newClient                   newClientFunc
	newClientWithoutCredentials func(credentials.Provider) (secrethub.ClientInterface, error)
	newClientWithSetupCode      func(string) (secrethub.ClientInterface, error)
	credentialStore             CredentialConfig
	progressPrinter             progress.Printer
}

// NewInitCommand creates a new InitCommand.
func NewInitCommand(io ui.IO, newClient newClientFunc, newClientWithoutCredentials func(credentials.Provider) (secrethub.ClientInterface, error), newClientWithSetupCode func(string) (secrethub.ClientInterface, error), credentialStore CredentialConfig) *InitCommand {
	return &InitCommand{
		io:                          io,
		newClient:                   newClient,
		newClientWithoutCredentials: newClientWithoutCredentials,
		newClientWithSetupCode:      newClientWithSetupCode,
		credentialStore:             credentialStore,
		progressPrinter:             progress.NewPrinter(io.Output(), 500*time.Millisecond),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *InitCommand) Register(r command.Registerer) {
	clause := r.Command("init", "Initialize the SecretHub client for first use on this device.")
	clause.Flag("backup-code", "The backup code used to restore an existing account to this device.").StringVar(&cmd.backupCode)
	clause.Flag("setup-code", "The setup code used to configure the CLI to use an account created on the website.").StringVar(&cmd.setupCode)
	registerForceFlag(clause).BoolVar(&cmd.force)

	command.BindAction(clause, cmd.Run)
}

type InitMode int

const (
	InitModeSignup InitMode = iota + 1
	InitModeBackupCode
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
				"Signup for a new account",
				"Use a setup code to set up an account created through the website",
				"Use a backup code to recover an existing account",
			}, 3)
		if err != nil {
			return err
		}

		switch option {
		case 0:
			mode = InitModeSignup
		case 1:
			mode = InitModeSetupCode
		case 2:
			mode = InitModeBackupCode
		}
	}

	switch mode {
	case InitModeSignup:
		signupCommand := SignUpCommand{
			io:              cmd.io,
			newClient:       cmd.newClient,
			credentialStore: cmd.credentialStore,
			progressPrinter: cmd.progressPrinter,
			force:           cmd.force,
		}
		return signupCommand.Run()
	case InitModeSetupCode:
		setupCode := cmd.setupCode
		if setupCode == "" {
			var err error
			setupCode, err = ui.AskAndValidate(cmd.io, "What is your setup code?\n", 3, api.ValidateSetupCode)
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(
			cmd.io.Output(),
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

		fmt.Fprint(cmd.io.Output(), "Setting up your account...")
		cmd.progressPrinter.Start()

		client, err := cmd.newClientWithSetupCode(setupCode)
		if err != nil {
			return err
		}

		credential := credentials.CreateKey()
		_, err = client.Credentials().Create(credential, "")
		if err != nil {
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

		client, err = cmd.newClient()
		if err != nil {
			return err
		}

		me, err := client.Me().GetUser()
		if err != nil {
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

		client, err := cmd.newClientWithoutCredentials(credentials.UseBackupCode(backupCode))
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

		deviceName := ""
		question := "What is the name of this device?"
		hostName, err := os.Hostname()
		if err == nil {
			deviceName, err = ui.AskWithDefault(cmd.io, question, hostName)
			if err != nil {
				return err
			}
		} else {
			deviceName, err = ui.Ask(cmd.io, question)
			if err != nil {
				return err
			}
		}

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
