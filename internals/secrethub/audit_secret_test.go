package secrethub

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestAuditSecretCommand_run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd AuditCommand
		err error
		out string
	}{
		"create secret event": {
			cmd: AuditCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							ExistsFunc: func(_ string) (bool, error) {
								return false, nil
							},
						},
						SecretService: &fakeclient.SecretService{
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
										LoggedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.Local),
										Subject: api.AuditSubject{
											Type: "secret",
										},
										IPAddress: "127.0.0.1",
									},
								},
							},
						},
					}, nil
				},
				format:     formatTable,
				perPage:    20,
				maxResults: -1,
				terminalWidth: func(_ int) (int, error) {
					return 46, nil
				},
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			out: "AUTHOR      EVENT       IP ADDRESS  DATE      \n" +
				"developer   create.sec  127.0.0.1   2018-01-01\n" +
				"            ret                     T01:01:01+\n" +
				"                                    01:00     \n",
		},
		"0 events": {
			cmd: AuditCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							ExistsFunc: func(_ string) (bool, error) {
								return false, nil
							},
						},
						SecretService: &fakeclient.SecretService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{},
							},
						},
					}, nil
				},
				format:     formatTable,
				perPage:    20,
				maxResults: -1,
				terminalWidth: func(_ int) (int, error) {
					return 46, nil
				},
			},
			out: "",
		},
		"error secret version": {
			cmd: AuditCommand{
				path:    "namespace/repo/secret:1",
				format:  formatTable,
				perPage: 20,
			},
			err: ErrCannotAuditSecretVersion,
		},
		"client creation error": {
			cmd: AuditCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientInterface, error) {
					return nil, ErrCannotFindHomeDir()
				},
				format:  formatTable,
				perPage: 20,
			},
			err: ErrCannotFindHomeDir(),
		},
		"error can not audit dir": {
			cmd: AuditCommand{
				path: "namespace/repo/dir",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							ExistsFunc: func(_ string) (bool, error) {
								return true, nil
							},
						},
						SecretService: &fakeclient.SecretService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Err: api.ErrSecretNotFound,
							},
						},
					}, nil
				},
				format:  formatTable,
				perPage: 20,
			},
			err: ErrCannotAuditDir,
		},
		"other list audit events error": {
			cmd: AuditCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							ExistsFunc: func(_ string) (bool, error) {
								return false, nil
							},
						},
						SecretService: &fakeclient.SecretService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Err: testError,
							},
						},
					}, nil
				},
				format:     formatTable,
				perPage:    20,
				maxResults: -1,
				terminalWidth: func(_ int) (int, error) {
					return 46, nil
				},
			},
			err: testError,
			out: "",
		},
		"invalid audit actor": {
			cmd: AuditCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							ExistsFunc: func(_ string) (bool, error) {
								return false, nil
							},
						},
						SecretService: &fakeclient.SecretService{
							AuditEventIterator: &fakeclient.AuditEventIterator{
								Events: []api.Audit{
									{},
								},
							},
						},
					}, nil
				},
				format:     formatTable,
				perPage:    20,
				maxResults: -1,
				terminalWidth: func(int) (int, error) {
					return 83, nil
				},
			},
			err: ErrInvalidAuditActor,
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
