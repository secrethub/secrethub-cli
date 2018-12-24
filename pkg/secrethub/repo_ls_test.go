package secrethub

import (
	"testing"

	"time"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/secrethub/fakes"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoLSCommand_run(t *testing.T) {
	testutil.Unit(t)

	testTime := time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC)
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          RepoLSCommand
		newClientErr error
		repoService  fakeclient.RepoService
		out          string
		err          error
	}{
		"success two repos": {
			cmd: RepoLSCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			repoService: fakeclient.RepoService{
				MineLister: fakeclient.RepoMineLister{
					ReturnsRepos: []*api.Repo{
						{
							Owner:     "dev1",
							Name:      "repository",
							Status:    api.StatusOK,
							CreatedAt: &testTime,
						},
						{
							Owner:     "dev2",
							Name:      "applicationname",
							Status:    api.StatusOK,
							CreatedAt: &testTime,
						},
					},
				},
			},
			out: "NAME                  STATUS  CREATED\n" +
				"dev1/repository       ok      2018-01-01T01:01:01+01:00\n" +
				"dev2/applicationname  ok      2018-01-01T01:01:01+01:00\n",
		},
		"success two repos quiet": {
			cmd: RepoLSCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
				quiet: true,
			},
			repoService: fakeclient.RepoService{
				MineLister: fakeclient.RepoMineLister{
					ReturnsRepos: []*api.Repo{
						{
							Owner:     "dev1",
							Name:      "repository",
							Status:    api.StatusOK,
							CreatedAt: &testTime,
						},
						{
							Owner:     "dev2",
							Name:      "applicationname",
							Status:    api.StatusOK,
							CreatedAt: &testTime,
						},
					},
				},
			},
			out: "dev1/repository\n" +
				"dev2/applicationname\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"repo mine error": {
			repoService: fakeclient.RepoService{
				MineLister: fakeclient.RepoMineLister{
					Err: testErr,
				},
			},
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.Client, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.Client, error) {
					return fakeclient.Client{
						RepoService: &tc.repoService,
					}, nil
				}
			}

			// Run
			err := tc.cmd.run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
