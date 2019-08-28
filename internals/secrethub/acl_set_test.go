package secrethub

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLSetCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd       ACLSetCommand
		in        string
		askErr    error
		err       error
		stdout    string
		promptOut string
	}{
		"success": {
			cmd: ACLSetCommand{
				accountName: "dev1",
				permission:  api.PermissionRead,
				path:        "namespace/repo/dir",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						AccessRuleService: &fakeclient.AccessRuleService{},
					}, nil
				},
			},
			in: "y",
			stdout: "Setting access rule for dev1 at namespace/repo/dir with read\n" +
				"Access rule set!\n",
			promptOut: "[WARNING] This gives dev1 read rights on all directories and secrets contained in namespace/repo/dir. " +
				"Are you sure you want to set this access rule? [y/N]: ",
		},
		"abort": {
			cmd: ACLSetCommand{
				accountName: "dev1",
				permission:  api.PermissionRead,
				path:        "namespace/repo/dir",
			},
			in:     "n",
			stdout: "Aborting.\n",
			promptOut: "[WARNING] This gives dev1 read rights on all directories and secrets contained in namespace/repo/dir. " +
				"Are you sure you want to set this access rule? [y/N]: ",
		},
		"client error": {
			cmd: ACLSetCommand{
				accountName: "dev1",
				permission:  api.PermissionRead,
				path:        "namespace/repo/dir",
				newClient: func() (secrethub.ClientAdapter, error) {
					return &fakeclient.Client{
						AccessRuleService: &fakeclient.AccessRuleService{
							Setter: fakeclient.AccessRuleSetter{
								Err: api.ErrAccessRuleNotFound,
							},
						},
					}, nil
				},
			},
			in:     "y",
			stdout: "Setting access rule for dev1 at namespace/repo/dir with read\n",
			promptOut: "[WARNING] This gives dev1 read rights on all directories and secrets contained in namespace/repo/dir. " +
				"Are you sure you want to set this access rule? [y/N]: ",
			err: api.ErrAccessRuleNotFound,
		},
		"ask error": {
			cmd: ACLSetCommand{
				accountName: "dev1",
				permission:  api.PermissionRead,
				path:        "namespace/repo/dir",
			},
			askErr: ui.ErrCannotAsk,
			err:    ui.ErrCannotAsk,
		},
		"client creation error": {
			cmd: ACLSetCommand{
				accountName: "dev1",
				permission:  api.PermissionRead,
				path:        "namespace/repo/dir",
				newClient: func() (secrethub.ClientAdapter, error) {
					return nil, testErr
				},
			},
			in:     "y",
			stdout: "Setting access rule for dev1 at namespace/repo/dir with read\n",
			promptOut: "[WARNING] This gives dev1 read rights on all directories and secrets contained in namespace/repo/dir. " +
				"Are you sure you want to set this access rule? [y/N]: ",
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			io.PromptErr = tc.askErr
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.stdout)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)

		})
	}
}
