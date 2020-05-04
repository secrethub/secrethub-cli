package secrethub

import (
	"bytes"
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLRmCommand_Run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd            ACLRmCommand
		newClientErr   error
		promptErr      error
		in             string
		argPath        api.Path
		argAccountName api.AccountName
		promptOut      string
		out            string
		deleteErr      error
		err            error
	}{
		"success force": {
			cmd: ACLRmCommand{
				force:       true,
				path:        "namespace/repo",
				accountName: "dev1",
			},
			argPath:        "namespace/repo",
			argAccountName: "dev1",
			out: "Removing access rule...\n" +
				"Removal complete! The access rule for dev1 on namespace/repo has been removed.\n",
		},
		"success": {
			cmd: ACLRmCommand{
				path:        "namespace/repo",
				accountName: "dev1",
			},
			in:             "y",
			argPath:        "namespace/repo",
			argAccountName: "dev1",
			promptOut: "[WARNING] This can impact the account's ability to read and/or modify secrets. " +
				"Are you sure you want to remove the access rule for dev1? [y/N]: ",
			out: "Removing access rule...\n" +
				"Removal complete! The access rule for dev1 on namespace/repo has been removed.\n",
		},
		"abort": {
			cmd: ACLRmCommand{
				path:        "namespace/repo",
				accountName: "dev1",
			},
			in: "n",
			promptOut: "[WARNING] This can impact the account's ability to read and/or modify secrets. " +
				"Are you sure you want to remove the access rule for dev1? [y/N]: ",
			out: "Aborting.\n",
		},
		"client creation error": {
			cmd: ACLRmCommand{
				force: true,
			},
			newClientErr: testError,
			err:          testError,
		},
		"prompt error": {
			promptErr: testError,
			err:       testError,
		},
		"delete error": {
			cmd: ACLRmCommand{
				force: true,
			},
			deleteErr: testError,
			out:       "Removing access rule...\n",
			err:       testError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			var argPath string
			var argAccountName string
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					AccessRuleService: &fakeclient.AccessRuleService{
						DeleteFunc: func(path string, accountName string) error {
							argPath = path
							argAccountName = accountName
							return tc.deleteErr
						},
					},
				}, tc.newClientErr
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, argPath, tc.argPath)
			assert.Equal(t, argAccountName, tc.argAccountName)
		})
	}
}
