package secrethub

import (
	"bytes"
	"errors"

	"github.com/secrethub/secrethub-go/internals/assert"

	"strings"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"

	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

func TestCredentialBackupCommand_Run(t *testing.T) {
	const confirmationString = "This will create a new backup code for Chucky. " +
		"This code can be used to obtain full access to your account.\n" +
		"Do you want to continue? [Y/n]: "

	var (
		testErr    = errors.New("test")
		failCreate = func(creator credentials.Creator) *api.Credential {
			return &api.Credential{}
		}
		succeedCreate = func(creator credentials.Creator) *api.Credential {
			_ = creator.Create()
			return &api.Credential{}
		}
	)
	testCases := map[string]struct {
		cmd           CredentialBackupCommand
		promptOut     string
		out           string
		in            string
		shouldSucceed bool
		createFunc    func(creator credentials.Creator) *api.Credential
		createErr     error
		meErr         error
		newClientErr  error
		promptErr     error
		err           error
	}{
		"fail client error": {
			err:           testErr,
			newClientErr:  testErr,
			shouldSucceed: false,
		},
		"fail prompt error": {
			promptErr:     testErr,
			err:           testErr,
			shouldSucceed: false,
		},
		"fail me error": {
			err:           testErr,
			meErr:         testErr,
			shouldSucceed: false,
		},
		"fail abort": {
			cmd:           CredentialBackupCommand{},
			promptOut:     confirmationString,
			in:            "n",
			out:           "Aborting\n",
			shouldSucceed: false,
		},
		"fail create error": {
			cmd:           CredentialBackupCommand{},
			promptOut:     confirmationString,
			in:            "y",
			err:           testErr,
			createErr:     testErr,
			createFunc:    succeedCreate,
			shouldSucceed: false,
		},
		"fail no account key": {
			cmd:           CredentialBackupCommand{},
			promptOut:     confirmationString,
			in:            "y",
			err:           errors.New("backup code has not yet been generated"),
			createFunc:    failCreate,
			shouldSucceed: false,
		},
		"success": {
			cmd:       CredentialBackupCommand{},
			promptOut: confirmationString,
			in:        "y",
			out: "This is your backup code: \n%s\n" + "Write it down and store it in a safe location! " +
				"You can restore your account by running `secrethub init`.",
			createFunc:    succeedCreate,
			shouldSucceed: true,
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
					MeService: &fakeclient.MeService{
						GetUserFunc: func() (*api.User, error) {
							return &api.User{
								Username: "Chucky",
							}, tc.meErr
						},
					},
					CredentialService: &fakeclient.CredentialService{
						CreateFunc: func(creator credentials.Creator, s string) (*api.Credential, error) {
							return tc.createFunc(creator), tc.createErr
						},
					},
				}
				return client, tc.newClientErr
			}

			err := tc.cmd.Run()

			assert.Equal(t, err, tc.err)
			// Since at the moment there is no way to retrieve the generated backup code
			// we should just compare the beginning and the end of the message with the
			// expected values, omitting, in this way, any assertion on the backup code.
			if !tc.shouldSucceed {
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
