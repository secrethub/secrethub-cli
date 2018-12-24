package secrethub

import (
	"testing"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgSetRoleCommand_Run(t *testing.T) {
	testutil.Unit(t)

	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgSetRoleCommand
		service      fakeclient.OrgMemberService
		newClientErr error
		ArgOrgName   api.OrgName
		ArgUsername  string
		ArgRole      string
		out          string
		err          error
	}{
		"success": {
			cmd: OrgSetRoleCommand{
				username: "dev1",
				orgName:  "company",
				role:     api.OrgRoleMember,
			},
			service: fakeclient.OrgMemberService{
				Updater: fakeclient.OrgMemberUpdater{
					ReturnsOrgMember: &api.OrgMember{
						User: &api.User{
							Username: "dev1",
						},
						Role: api.OrgRoleMember,
					},
				},
			},
			ArgOrgName:  "company",
			ArgUsername: "dev1",
			ArgRole:     api.OrgRoleMember,
			out: "Setting role...\n" +
				"Set complete! The user dev1 is member of the company organization.\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"update org member error": {
			service: fakeclient.OrgMemberService{
				Updater: fakeclient.OrgMemberUpdater{
					Err: testErr,
				},
			},
			out: "Setting role...\n",
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					OrgService: &fakeclient.OrgService{
						MemberService: &tc.service,
					},
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
			testutil.Compare(t, tc.service.Updater.ArgOrgName, tc.ArgOrgName)
			testutil.Compare(t, tc.service.Updater.ArgUsername, tc.ArgUsername)
			testutil.Compare(t, tc.service.Updater.ArgRole, tc.ArgRole)
		})
	}
}
