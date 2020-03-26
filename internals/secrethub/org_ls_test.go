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

func TestOrgLsCommand_run(t *testing.T) {
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
				ListMineFunc: func() ([]*api.Org, error) {
					return []*api.Org{
						{
							Name:      "company1",
							CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
						{
							Name:      "company2",
							CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
					}, nil
				},
				MembersService: &fakeclient.OrgMemberService{
					ListFunc: func(org string) ([]*api.OrgMember, error) {
						return []*api.OrgMember{
							{},
							{},
							{},
						}, nil
					},
				},
			},
			repoService: fakeclient.RepoService{
				ListFunc: func(namespace string) ([]*api.Repo, error) {
					return []*api.Repo{
						{},
						{},
					}, nil
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
				ListMineFunc: func() ([]*api.Org, error) {
					return []*api.Org{
						{
							Name: "company1",
						},
						{
							Name: "company2",
						},
					}, nil
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
				ListMineFunc: func() ([]*api.Org, error) {
					return nil, testErr
				},
			},
			err: testErr,
		},
		"list org member error": {
			service: fakeclient.OrgService{
				ListMineFunc: func() ([]*api.Org, error) {
					return []*api.Org{
						{},
					}, nil
				},
				MembersService: &fakeclient.OrgMemberService{
					ListFunc: func(org string) ([]*api.OrgMember, error) {
						return nil, testErr
					},
				},
			},
			err: testErr,
		},
		"list repos error": {
			service: fakeclient.OrgService{
				ListMineFunc: func() ([]*api.Org, error) {
					return []*api.Org{
						{},
					}, nil
				},
				MembersService: &fakeclient.OrgMemberService{
					ListFunc: func(org string) ([]*api.OrgMember, error) {
						return nil, nil
					},
				},
			},
			repoService: fakeclient.RepoService{
				ListFunc: func(namespace string) ([]*api.Repo, error) {
					return nil, testErr
				},
			},
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
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
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
