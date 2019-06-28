package secrethub

import (
	"errors"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestAuditRepoCommand_run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd AuditRepoCommand
		err error
		out string
	}{
		"0 events": {
			cmd: AuditRepoCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.Client, error) {
					return &fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{},
						},
					}, nil
				},
			},
			out: "AUTHOR    EVENT    EVENT SUBJECT    IP ADDRESS    DATE\n",
		},
		"create repo event": {
			cmd: AuditRepoCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{
								ReturnsAuditEvents: []*api.Audit{
									{
										Action: "create",
										Actor: api.AuditActor{
											Type: "user",
											User: &api.User{
												Username: "developer",
											},
										},
										LoggedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
										Subject: api.AuditSubject{
											Type: "repo",
											Repo: &api.Repo{
												Name: "repo",
											},
										},
										IPAddress: "127.0.0.1",
									},
								},
								Err: nil,
							},
						},
					}, nil
				},
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			out: "AUTHOR       EVENT          EVENT SUBJECT    IP ADDRESS    DATE\n" +
				"developer    create.repo    repo             127.0.0.1     2018-01-01T01:01:01+01:00\n",
		},
		"client creation error": {
			cmd: AuditRepoCommand{
				newClient: func() (secrethub.Client, error) {
					return nil, ErrCannotFindHomeDir()
				},
			},
			err: ErrCannotFindHomeDir(),
		},
		"list audit events error": {
			cmd: AuditRepoCommand{
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{
								Err: testError,
							},
						},
					}, nil
				},
			},
			err: testError,
		},
		"get dir error": {
			cmd: AuditRepoCommand{
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							TreeGetter: fakeclient.TreeGetter{
								Err: testError,
							},
						},
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{},
						},
					}, nil
				},
			},
			err: testError,
		},
		"invalid audit actor": {
			cmd: AuditRepoCommand{
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{
								ReturnsAuditEvents: []*api.Audit{
									{},
								},
								Err: nil,
							},
						},
					}, nil
				},
			},
			err: ErrInvalidAuditActor,
			out: "",
		},
		"invalid audit subject": {
			cmd: AuditRepoCommand{
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							EventLister: fakeclient.RepoEventLister{
								ReturnsAuditEvents: []*api.Audit{
									{
										Actor: api.AuditActor{
											Type: "user",
											User: &api.User{
												Username: "developer",
											},
										},
									},
								},
								Err: nil,
							},
						},
					}, nil
				},
			},
			err: ErrInvalidAuditSubject,
			out: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
