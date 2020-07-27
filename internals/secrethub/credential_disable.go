package secrethub

import (
	"errors"
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/internals/api"
)

var ErrForceNoFingerprint = errors.New("fingerprint argument must be set when using --force")

// CredentialDisableCommand is a command that allows to disable an existing credential.
type CredentialDisableCommand struct {
	io          ui.IO
	force       bool
	fingerprint string
	newClient   newClientFunc
}

// NewCredentialDisableCommand creates a new command for disabling credentials.
func NewCredentialDisableCommand(io ui.IO, newClient newClientFunc) *CredentialDisableCommand {
	return &CredentialDisableCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CredentialDisableCommand) Register(r command.Registerer) {
	clause := r.Command("disable", "Disable a credential for usage on SecretHub.")

	fingerprintHelp := fmt.Sprintf("Fingerprint of the credential to disable. At least the first %d characters must be entered.", api.ShortCredentialFingerprintMinimumLength)
	clause.Arg("fingerprint", fingerprintHelp).StringVar(&cmd.fingerprint)

	registerForceFlag(clause).BoolVar(&cmd.force)

	command.BindAction(clause, cmd.Run)
}

// Run disables an existing credential.
func (cmd *CredentialDisableCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fingerprint := cmd.fingerprint
	if fingerprint == "" {
		if cmd.force {
			return ErrForceNoFingerprint
		}
		fingerprint, err = ui.AskAndValidate(cmd.io, "What is the fingerprint of the credential you want to disable? ", 3, api.ValidateShortCredentialFingerprint)
		if err != nil {
			return err
		}
	}

	err = api.ValidateShortCredentialFingerprint(fingerprint)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(),
		"A disabled credential can no longer be used to access SecretHub. "+
			"This process can currently not be reversed.")

	if !cmd.force {
		ok, err := ui.AskYesNo(cmd.io, fmt.Sprintf("Are you sure you want to disable the credential with fingerprint %s?", fingerprint), ui.DefaultNo)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}
	}

	err = client.Credentials().Disable(fingerprint)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Credential disabled.")

	return nil
}
