package secrethub

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestAuditRepoCommand_run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd AuditCommand
		err error
		out string
	}{
		"0 events": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return &fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{},
							},
						},
					}, nil
				},
				terminalWidth: func(int) (int, error) {
					return 83, nil
				},
				format:  formatTable,
				perPage: 20,
			},
			out: "AUTHOR           EVENT            EVENT SUBJECT    IP ADDRESS       DATE           \n",
		},
		"create repo event": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{
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
							},
						},
					}, nil
				},
				format:  formatTable,
				perPage: 20,
				terminalWidth: func(_ int) (int, error) {
					return 83, nil
				},
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			out: "AUTHOR           EVENT            EVENT SUBJECT    IP ADDRESS       DATE           \n" +
				"developer        create.repo      repo             127.0.0.1        2018-01-01T01:0\n" +
				"                                                                    1:01+01:00     \n",
		},
		"client creation error": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return nil, ErrCannotFindHomeDir()
				},
				format:  formatTable,
				perPage: 20,
			},
			err: ErrCannotFindHomeDir(),
		},
		"list audit events error": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						RepoService: &fakeclient.RepoService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Err: testError,
							},
						},
						DirService: &fakeclient.DirService{
							TreeGetter: fakeclient.TreeGetter{},
						},
					}, nil
				},
				format:  formatTable,
				perPage: 20,
				terminalWidth: func(int) (int, error) {
					return 83, nil
				},
			},
			err: testError,
			out: "AUTHOR           EVENT            EVENT SUBJECT    IP ADDRESS       DATE           \n",
		},
		"get dir error": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
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
				format:  formatTable,
				perPage: 20,
			},
			err: testError,
		},
		"invalid audit actor": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{
									{
										Subject: api.AuditSubject{
											Type: api.AuditSubjectService,
											Service: &api.Service{
												ServiceID: "<service id>",
											},
										},
									},
								},
							},
						},
					}, nil
				},
				format:  formatTable,
				perPage: 20,
				terminalWidth: func(int) (int, error) {
					return 83, nil
				},
			},
			err: ErrInvalidAuditActor,
			out: "AUTHOR           EVENT            EVENT SUBJECT    IP ADDRESS       DATE           \n",
		},
		"invalid audit subject": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{},
						RepoService: &fakeclient.RepoService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{
									{
										Actor: api.AuditActor{
											Type: "user",
											User: &api.User{
												Username: "developer",
											},
										},
									},
								},
							},
						},
					}, nil
				},
				format:  formatTable,
				perPage: 20,
				terminalWidth: func(int) (int, error) {
					return 83, nil
				},
			},
			err: ErrInvalidAuditSubject,
			out: "AUTHOR           EVENT            EVENT SUBJECT    IP ADDRESS       DATE           \n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			buffer := bytes.Buffer{}
			tc.cmd.newPaginatedWriter = func(_ io.Writer) (pager, error) {
				return &fakes.Pager{Buffer: &buffer}, nil
			}

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, buffer.String(), tc.out)
		})
	}
}
