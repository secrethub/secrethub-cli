package secrethub

import (
	"bytes"
	"errors"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"gotest.tools/assert"
	"testing"
)

func TestCredentialBackupCommand_Run(t *testing.T) {
	var testError = errors.New("test")

	testCases := map[string]struct {
		cmd            CredentialBackupCommand
		promptOut      string
		out            string
		in             string
		meError        error
		newClientError error
		promptError    error
		err            error
	}{
		"fail-client-error": {
			err: testError,
			newClientError: testError,
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
					//MeService: ...
				}
				return client, tc.newClientError
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
