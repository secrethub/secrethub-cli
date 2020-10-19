package secrethub

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestCredentialBackupCommand_Run(t *testing.T) {
	const (
		confirmationString = "This will create a new backup code for Chucky. " +
			"This code can be used to obtain full access to your account.\n" +
			"Do you want to continue? [Y/n]: "
		backupPrefix = "This is your backup code: \n"
		backupSuffix = "\nWrite it down and store it in a safe location! You can restore your account by running `secrethub init`.\n"
	)

	testErr := errors.New("test")
	failCreate := func(creator credentials.Creator) *api.Credential {
		return &api.Credential{}
	}
	succeedCreate := func(creator credentials.Creator) *api.Credential {
		_ = creator.Create()
		return &api.Credential{}
	}
	testCases := map[string]struct {
		cmd               CredentialBackupCommand
		in                string
		shouldSucceed     bool
		createFunc        func(creator credentials.Creator) *api.Credential
		createErr         error
		meErr             error
		newClientErr      error
		promptErr         error
		expectedPromptOut string
		expectedOut       string
		expectedErr       error
	}{
		"fail client error": {
			expectedErr:   testErr,
			newClientErr:  testErr,
			shouldSucceed: false,
		},
		"fail prompt error": {
			promptErr:     testErr,
			expectedErr:   testErr,
			shouldSucceed: false,
		},
		"fail me error": {
			expectedErr:   testErr,
			meErr:         testErr,
			shouldSucceed: false,
		},
		"fail abort": {
			cmd:               CredentialBackupCommand{},
			expectedPromptOut: confirmationString,
			in:                "n",
			expectedOut:       "Aborting\n",
			shouldSucceed:     false,
		},
		"fail create error": {
			cmd:               CredentialBackupCommand{},
			expectedPromptOut: confirmationString,
			in:                "y",
			expectedErr:       testErr,
			createErr:         testErr,
			createFunc:        succeedCreate,
			shouldSucceed:     false,
		},
		"fail no account key": {
			cmd:               CredentialBackupCommand{},
			expectedPromptOut: confirmationString,
			in:                "y",
			expectedErr:       errors.New("backup code has not yet been generated"),
			createFunc:        failCreate,
			shouldSucceed:     false,
		},
		"success": {
			cmd:               CredentialBackupCommand{},
			expectedPromptOut: confirmationString,
			in:                "y",
			expectedOut: "This is your backup code: \n%s\n" + "Write it down and store it in a safe location! " +
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

			assert.Equal(t, err, tc.expectedErr)
			// Since at the moment there is no way to retrieve the generated backup code
			// we should just compare the beginning and the end of the message with the
			// expected values, omitting, in this way, any assertion on the backup code.
			if !tc.shouldSucceed {
				assert.Equal(t, io.Out.String(), tc.expectedOut)
			} else {
				beginning := strings.HasPrefix(io.Out.String(), backupPrefix)
				end := strings.HasSuffix(io.Out.String(), backupSuffix)
				if !beginning && !end {
					t.Errorf("The output did not match the expected format. " + io.Out.String())
				}
				credentialValue := strings.TrimSuffix(strings.TrimPrefix(io.Out.String(), backupPrefix), backupSuffix)
				assert.Equal(t, len(credentialValue), 71)
			}
			assert.Equal(t, io.PromptOut.String(), tc.expectedPromptOut)
		})
	}
}
