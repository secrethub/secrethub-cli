package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgInspectCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgInspectCommand
		orgService   fakeclient.OrgService
		repoService  fakeclient.RepoService
		newClientErr error
		out          string
		err          error
	}{
		"success": {
			cmd: OrgInspectCommand{
				name: "company",
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+00:00",
				},
			},
			orgService: fakeclient.OrgService{
				Getter: fakeclient.OrgGetter{
					ReturnsOrg: &api.Org{
						Name:        "company",
						CreatedAt:   time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						Description: "description of the company.",
					},
				},
				MemberService: &fakeclient.OrgMemberService{
					Lister: fakeclient.OrgMemberLister{
						ReturnsMembers: []*api.OrgMember{
							{
								Role: api.OrgRoleAdmin,
								User: &api.User{
									Username: "dev1",
								},
								CreatedAt:     time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
								LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
							},
							{
								Role: api.OrgRoleMember,
								User: &api.User{
									Username: "dev2",
								},
								CreatedAt:     time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
								LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
							},
						},
					},
				},
			},
			repoService: fakeclient.RepoService{
				Lister: fakeclient.RepoLister{
					ReturnsRepos: []*api.Repo{
						{
							Name: "application1",
						},
						{
							Name: "application2",
						},
					},
				},
			},
			out: "{\n" +
				"    \"Name\": \"company\",\n" +
				"    \"Description\": \"description of the company.\",\n" +
				"    \"CreatedAt\": \"2018-01-01T01:01:01+00:00\",\n" +
				"    \"MemberCount\": 2,\n" +
				"    \"Members\": [\n" +
				"        {\n" +
				"            \"Username\": \"dev1\",\n" +
				"            \"Role\": \"admin\",\n" +
				"            \"CreatedAt\": \"2018-01-01T01:01:01+00:00\",\n" +
				"            \"LastChangedAt\": \"2018-01-01T01:01:01+00:00\"\n" +
				"        },\n" +
				"        {\n" +
				"            \"Username\": \"dev2\",\n" +
				"            \"Role\": \"member\",\n" +
				"            \"CreatedAt\": \"2018-01-01T01:01:01+00:00\",\n" +
				"            \"LastChangedAt\": \"2018-01-01T01:01:01+00:00\"\n" +
				"        }\n" +
				"    ],\n" +
				"    \"RepoCount\": 2,\n" +
				"    \"Repos\": [\n" +
				"        \"/application1\",\n" +
				"        \"/application2\"\n" +
				"    ]\n" +
				"}\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"get org error": {
			orgService: fakeclient.OrgService{
				Getter: fakeclient.OrgGetter{
					Err: testErr,
				},
			},
			err: testErr,
		},
		"list org members error": {
			orgService: fakeclient.OrgService{
				MemberService: &fakeclient.OrgMemberService{
					Lister: fakeclient.OrgMemberLister{
						Err: testErr,
					},
				},
			},
			err: testErr,
		},
		"list repos error": {
			orgService: fakeclient.OrgService{
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
			tc.cmd.newClient = func() (secrethub.ClientAdapter, error) {
				return fakeclient.Client{
					OrgService:  &tc.orgService,
					RepoService: &tc.repoService,
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
