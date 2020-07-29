package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
	"gotest.tools/assert"
)

func TestCredentialDisableCommand_Run(t *testing.T) {

	var testError = errors.New("test")

	testCases := map[string]struct {
		cmd            CredentialDisableCommand
		force          bool
		promptOut      string
		out            string
		in             string
		disableError   error
		newClientError error
		promptError    error
		err            error
	}{
		"fail-force-no-credential": {
			cmd: CredentialDisableCommand{
				fingerprint: "",
				force:       true,
			},
			err: ErrForceNoFingerprint,
		},
		"fail-prompt-error": {
			promptError: testError,
			err:         testError,
		},
		"fail-empty-credential-3-times": {
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
		"fail-abort": {
			cmd: CredentialDisableCommand{
				fingerprint: "62542d734d7f3627",
				force:       false,
			},
			promptOut: "Are you sure you want to disable the credential with fingerprint 62542d734d7f3627? [y/N]: ",
			in:        "n",
			out:       "A disabled credential can no longer be used to access SecretHub. This process can currently not be reversed.\nAborting.\n",
		},
		"fail-client-error": {
			newClientError: testError,
			err:            testError,
		},
		"fail-force-wrong-credential": {
			cmd: CredentialDisableCommand{
				fingerprint: "BillyBoy",
				force:       true,
			},
			err: api.ErrInvalidFingerprint,
		},
		"fail-too-short-fingerprint": {
			cmd: CredentialDisableCommand{
				fingerprint: "6254",
				force:       true,
			},
			err: api.ErrTooShortFingerprint,
		},
		"succeed-force": {
			cmd: CredentialDisableCommand{
				fingerprint: "62542d734d7f3627",
				force:       true,
			},
			out: "A disabled credential can no longer be used to access SecretHub. " +
				"This process can currently not be reversed.\nCredential disabled.\n",
		},
		"succeed-no-force": {
			cmd: CredentialDisableCommand{
				fingerprint: "62542d734d7f3627",
				force:       false,
			},
			promptOut: "Are you sure you want to disable the credential with fingerprint 62542d734d7f3627? [y/N]: ",
			in:        "y",
			out:       "A disabled credential can no longer be used to access SecretHub. This process can currently not be reversed.\nCredential disabled.\n",
		},
		"fail-disable-error": {
			cmd: CredentialDisableCommand{
				fingerprint: "62542d734d7f3628",
				force:       true,
			},
			out: "A disabled credential can no longer be used to access SecretHub. " +
				"This process can currently not be reversed.\n",
			disableError: testError,
			err:          testError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptError
			tc.cmd.io = io

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				client := fakeclient.Client{
					CredentialService: &fakeclient.CredentialService{
						DisableFunc: func(fingerprint string) error {
							if fingerprint == "62542d734d7f3627" {
								return nil
							}
							return tc.disableError
						},
					},
				}
				return &client, tc.newClientError
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
