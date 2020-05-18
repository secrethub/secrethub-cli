package secrethub

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"
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
						DirService: &fakeclient.DirService{
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, nil
							},
						},
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
			out: "",
		},
		"create repo event": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, nil
							},
						},
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
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, nil
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
			err: testError,
			out: "",
		},
		"get dir error": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, testError
							},
						},
						RepoService: &fakeclient.RepoService{
							ListEventsFunc: func(path string, subjectTypes api.AuditSubjectTypeList) ([]*api.Audit, error) {
								return nil, nil
							},
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
						DirService: &fakeclient.DirService{
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, nil
							},
						},
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
			out: "",
		},
		"invalid audit subject": {
			cmd: AuditCommand{
				path: "namespace/repo",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
								return nil, nil
							},
						},
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
			out: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			buffer := bytes.Buffer{}
			tc.cmd.newPaginatedWriter = func(_ io.Writer) (io.WriteCloser, error) {
				return &fakes.Pager{Buffer: &buffer}, nil
			}
			tc.cmd.io = fakeui.NewIO(t)

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, buffer.String(), tc.out)
		})
	}
}
