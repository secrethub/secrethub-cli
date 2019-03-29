package secrethub

import (
	"errors"
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"bytes"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLRmCommand_Run(t *testing.T) {
	testError := errio.Error(errors.New("test error"))

	cases := map[string]struct {
		cmd            ACLRmCommand
		deleter        fakeclient.AccessRuleDeleter
		newClientErr   error
		promptErr      error
		in             string
		argPath        api.Path
		argAccountName api.AccountName
		promptOut      string
		out            string
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
			deleter: fakeclient.AccessRuleDeleter{
				Err: testError,
			},
			out: "Removing access rule...\n",
			err: testError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			deleter := &tc.deleter

			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					AccessRuleService: &fakeclient.AccessRuleService{
						Deleter: deleter,
					},
				}, tc.newClientErr
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, deleter.ArgPath, tc.argPath)
			assert.Equal(t, deleter.ArgAccountName, tc.argAccountName)
		})
	}
}
