package secrethub

import (
	"testing"

	"time"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	fakes "github.com/keylockerbv/secrethub-cli/pkg/secrethub/fakes"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgLsCommand_run(t *testing.T) {
	testutil.Unit(t)

	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgLsCommand
		service      fakeclient.OrgService
		repoService  fakeclient.RepoService
		newClientErr error
		out          string
		err          error
	}{
		"success": {
			cmd: OrgLsCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+00:00",
				},
			},
			service: fakeclient.OrgService{
				MineLister: fakeclient.OrgMineLister{
					ReturnsOrgs: []*api.Org{
						{
							Name:      "company1",
							CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
						{
							Name:      "company2",
							CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
					},
				},
				MemberService: &fakeclient.OrgMemberService{
					Lister: fakeclient.OrgMemberLister{
						ReturnsMembers: []*api.OrgMember{
							{},
							{},
							{},
						},
					},
				},
			},
			repoService: fakeclient.RepoService{
				Lister: fakeclient.RepoLister{
					ReturnsRepos: []*api.Repo{
						{},
						{},
					},
				},
			},
			out: "NAME      REPOS  USERS  CREATED\n" +
				"company1  2      3      2018-01-01T01:01:01+00:00\n" +
				"company2  2      3      2018-01-01T01:01:01+00:00\n",
		},
		"success quiet": {
			cmd: OrgLsCommand{
				quiet: true,
			},
			service: fakeclient.OrgService{
				MineLister: fakeclient.OrgMineLister{
					ReturnsOrgs: []*api.Org{
						{
							Name: "company1",
						},
						{
							Name: "company2",
						},
					},
				},
			},
			out: "company1\n" +
				"company2\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"orgs mine error": {
			service: fakeclient.OrgService{
				MineLister: fakeclient.OrgMineLister{
					Err: testErr,
				},
			},
			err: testErr,
		},
		"list org member error": {
			service: fakeclient.OrgService{
				MineLister: fakeclient.OrgMineLister{
					ReturnsOrgs: []*api.Org{
						{},
					},
				},
				MemberService: &fakeclient.OrgMemberService{
					Lister: fakeclient.OrgMemberLister{
						Err: testErr,
					},
				},
			},
			err: testErr,
		},
		"list repos error": {
			service: fakeclient.OrgService{
				MineLister: fakeclient.OrgMineLister{
					ReturnsOrgs: []*api.Org{
						{},
					},
				},
				MemberService: &fakeclient.OrgMemberService{},
			},
			repoService: fakeclient.RepoService{
				Lister: fakeclient.RepoLister{
					Err: testErr,
				},
			},
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					OrgService:  &tc.service,
					RepoService: &tc.repoService,
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
