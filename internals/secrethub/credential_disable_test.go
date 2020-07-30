package secrethub

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestCredentialDisableCommand_Run(t *testing.T) {

	var testErr = errors.New("test")
	const (
		validFingerprint = "62542d734d7f3627"
		warningMessage   = "A disabled credential can no longer be used to access SecretHub. " +
			"This process can currently not be reversed.\n"
	)
	testCases := map[string]struct {
		cmd          CredentialDisableCommand
		force        bool
		promptOut    string
		out          string
		in           string
		disableErr   error
		newClientErr error
		promptErr    error
		err          error
	}{
		"fail force no credential": {
			cmd: CredentialDisableCommand{
				fingerprint: "",
				force:       true,
			},
			err: errors.New("fingerprint argument must be set when using --force"),
		},
		"fail prompt error": {
			promptErr: testErr,
			err:       testErr,
		},
		"fail empty credential 3 times": {
			cmd: CredentialDisableCommand{
				fingerprint: "",
				force:       false,
			},
			promptOut: "What is the fingerprint of the credential you want to disable? \n" +
				"Invalid input: fingerprint is invalid (api.invalid_fingerprint) \n" +
				"Please try again.\n" +
				"What is the fingerprint of the credential you want to disable? \n" +
				"Invalid input: fingerprint is invalid (api.invalid_fingerprint) \n" +
				"Please try again.\n" +
				"What is the fingerprint of the credential you want to disable? \n" +
				"Invalid input: fingerprint is invalid (api.invalid_fingerprint) \n",
			err: api.ErrInvalidFingerprint,
		},
		"fail abort": {
			cmd: CredentialDisableCommand{
				fingerprint: validFingerprint,
				force:       false,
			},
			promptOut: fmt.Sprintf("Are you sure you want to disable the credential with fingerprint %s? [y/N]: ", validFingerprint),
			in:        "n",
			out:       warningMessage + "Aborting.\n",
		},
		"fail client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"fail force wrong credential": {
			cmd: CredentialDisableCommand{
				fingerprint: "BillyBoy",
				force:       true,
			},
			err: api.ErrInvalidFingerprint,
		},
		"fail too short fingerprint": {
			cmd: CredentialDisableCommand{
				fingerprint: "6254",
				force:       true,
			},
			err: api.ErrTooShortFingerprint,
		},
		"succeed force": {
			cmd: CredentialDisableCommand{
				fingerprint: validFingerprint,
				force:       true,
			},
			out: warningMessage + "Credential disabled.\n",
		},
		"succeed no force": {
			cmd: CredentialDisableCommand{
				fingerprint: validFingerprint,
				force:       false,
			},
			promptOut: fmt.Sprintf("Are you sure you want to disable the credential with fingerprint %s? [y/N]: ", validFingerprint),
			in:        "y",
			out:       warningMessage + "Credential disabled.\n",
		},
		"fail disable error": {
			cmd: CredentialDisableCommand{
				fingerprint: "62542d734d7f3628",
				force:       true,
			},
			out:        warningMessage,
			disableErr: testErr,
			err:        testErr,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				client := fakeclient.Client{
					CredentialService: &fakeclient.CredentialService{
						DisableFunc: func(fingerprint string) error {
							if fingerprint == validFingerprint {
								return nil
							}
							return tc.disableErr
						},
					},
				}
				return &client, tc.newClientErr
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
