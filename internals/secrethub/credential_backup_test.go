package secrethub

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"

	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"gotest.tools/assert"
)

func TestCredentialBackupCommand_Run(t *testing.T) {
	var testError = errors.New("test")

	testCases := map[string]struct {
		cmd            CredentialBackupCommand
		promptOut      string
		out            string
		in             string
		username       string
		createError    error
		meError        error
		newClientError error
		promptError    error
		err            error
	}{
		"fail-client-error": {
			err:            testError,
			newClientError: testError,
		},
		"fail-me-error": {
			err:     testError,
			meError: testError,
		},
		"fail-abort": {
			cmd: CredentialBackupCommand{},
			promptOut: "This will create a new backup code for Chucky. " +
				"This code can be used to obtain full access to your account.\n" +
				"Do you want to continue? [Y/n]: ",
			in:  "n",
			out: "Aborting\n",
		},
		"fail-create-error": {
			cmd: CredentialBackupCommand{},
			promptOut: "This will create a new backup code for Chucky. " +
				"This code can be used to obtain full access to your account.\n" +
				"Do you want to continue? [Y/n]: ",
			in:          "y",
			err:         testError,
			createError: testError,
		},
		"fail-no-account-key": {
			cmd: CredentialBackupCommand{},
			promptOut: "This will create a new backup code for Chucky. " +
				"This code can be used to obtain full access to your account.\n" +
				"Do you want to continue? [Y/n]: ",
			in:  "y",
			err: errors.New("backup code has not yet been generated"),
		},
		/*

			This test fails, at the moment, since we need to find a way to inject a mock for the
			'backup.Code()' function, in order to avoid the check done on the http Client.
				"success": {
					cmd: CredentialBackupCommand{},
					promptOut: "This will create a new backup code for Chucky. " +
						"This code can be used to obtain full access to your account.\n" +
						"Do you want to continue? [Y/n]: ",
					in:  "y",
					out: "This is your backup code: \n%s\n" + "Write it down and store it in a safe location! "+
						"You can restore your account by running `secrethub init`.",
				},
		*/

	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptError
			tc.cmd.io = io

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				client := fakeclient.Client{
					MeService: &fakeclient.MeService{
						GetUserFunc: func() (*api.User, error) {
							return &api.User{
								Username: "Chucky",
							}, tc.meError
						},
					},
					CredentialService: &fakeclient.CredentialService{
						CreateFunc: func(creator credentials.Creator, s string) (*api.Credential, error) {
							return &api.Credential{}, tc.createError
						},
					},
				}
				return client, tc.newClientError
			}

			err := tc.cmd.Run()

			assert.Equal(t, fmt.Sprint(err), fmt.Sprint(tc.err))

			// Since at the moment there is no way to retrieve the generated backup code
			// we should just compare the beginning and the end of the message with the
			// expected values, omitting, in this way, any assertion on the backup code.
			if name != "success" {
				assert.Equal(t, io.Out.String(), tc.out)
			} else {
				splitIO := strings.Split(strings.TrimSuffix(io.Out.String(), "\n"), "\n")
				splitTC := strings.Split(strings.TrimSuffix(tc.out, "\n"), "\n")

				assert.Equal(t, splitIO[0], splitTC[0])
				assert.Equal(t, splitIO[len(splitIO)-1], splitTC[len(splitTC)-1])
			}
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
