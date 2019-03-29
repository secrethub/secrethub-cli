package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"time"

	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgListUsersCommand_run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd              OrgListUsersCommand
		service          fakeclient.OrgMemberService
		ArgListOrgMember api.OrgName
		newClientErr     error
		out              string
		err              error
	}{
		"success": {
			cmd: OrgListUsersCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+00:00",
				},
				orgName: "company",
			},
			service: fakeclient.OrgMemberService{
				Lister: fakeclient.OrgMemberLister{
					ReturnsMembers: []*api.OrgMember{
						{
							User: &api.User{
								Username: "dev1",
							},
							Role:          api.OrgRoleMember,
							LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
					},
				},
			},
			ArgListOrgMember: "company",
			out: "USER  ROLE    LAST CHANGED\n" +
				"dev1  member  2018-01-01T01:01:01+00:00\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"list org members error": {
			cmd: OrgListUsersCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+00:00",
				},
				orgName: "company",
			},
			service: fakeclient.OrgMemberService{
				Lister: fakeclient.OrgMemberLister{
					Err: testErr,
				},
			},
			ArgListOrgMember: "company",
			err:              testErr,
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
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, tc.service.Lister.ArgName, tc.ArgListOrgMember)
		})
	}
}
