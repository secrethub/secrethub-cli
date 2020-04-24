package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgListUsersCommand_run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd              OrgListUsersCommand
		listFunc         func(org string) ([]*api.OrgMember, error)
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
			listFunc: func(org string) ([]*api.OrgMember, error) {
				return []*api.OrgMember{
					{
						User: &api.User{
							Username: "dev1",
						},
						Role:          api.OrgRoleMember,
						LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
					},
				}, nil
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
			listFunc: func(org string) ([]*api.OrgMember, error) {
				return nil, testErr
			},
			ArgListOrgMember: "company",
			err:              testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var argOrg string

			// Setup
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					OrgService: &fakeclient.OrgService{
						MembersService: &fakeclient.OrgMemberService{
							ListFunc: func(org string) ([]*api.OrgMember, error) {
								argOrg = org
								return tc.listFunc(org)
							},
						},
					},
				}, tc.newClientErr
			}

			io := fakeui.NewIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
			assert.Equal(t, argOrg, tc.ArgListOrgMember)
		})
	}
}
