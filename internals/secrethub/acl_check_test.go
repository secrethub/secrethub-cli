package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLCheckCommand_Run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd           ACLCheckCommand
		newClientErr  error
		lister        fakeclient.AccessLevelLister
		listerArgPath api.Path
		out           string
		err           error
	}{
		"client creation error": {
			newClientErr: testError,
			err:          testError,
		},
		"success specific account": {
			cmd: ACLCheckCommand{
				accountName: "dev1",
				path:        "namespace/repo",
			},
			lister: fakeclient.AccessLevelLister{
				ReturnsAccessLevels: []*api.AccessLevel{
					{
						Account: &api.Account{
							Name: "dev1",
						},
						Permission: api.PermissionRead,
					},
					{
						Account: &api.Account{
							Name: "dev2",
						},
						Permission: api.PermissionWrite,
					},
				},
			},
			listerArgPath: "namespace/repo",
			out:           "read\n",
		},
		"success specific account no permission": {
			cmd: ACLCheckCommand{
				accountName: "dev1",
				path:        "namespace/repo",
			},
			lister: fakeclient.AccessLevelLister{
				ReturnsAccessLevels: []*api.AccessLevel{
					{
						Account: &api.Account{
							Name: "dev2",
						},
						Permission: api.PermissionWrite,
					},
				},
			},
			listerArgPath: "namespace/repo",
			out:           "none\n",
		},
		"success all accounts": {
			cmd: ACLCheckCommand{
				path: "namespace/repo",
			},
			lister: fakeclient.AccessLevelLister{
				ReturnsAccessLevels: []*api.AccessLevel{
					{
						Account: &api.Account{
							Name: "dev1",
						},
						Permission: api.PermissionRead,
					},
					{
						Account: &api.Account{
							Name: "dev2",
						},
						Permission: api.PermissionWrite,
					},
				},
			},
			listerArgPath: "namespace/repo",
			out: "PERMISSIONS    ACCOUNT\n" +
				"write          dev2\n" +
				"read           dev1\n",
		},
		"list error": {
			lister: fakeclient.AccessLevelLister{
				Err: testError,
			},
			err: testError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			lister := &tc.lister

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					AccessRuleService: &fakeclient.AccessRuleService{
						LevelLister: lister,
					},
				}, tc.newClientErr
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, lister.ArgPath, tc.listerArgPath)
		})
	}
}
