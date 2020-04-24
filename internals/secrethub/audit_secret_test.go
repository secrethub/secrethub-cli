package secrethub

import (
	"errors"
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
				perPage: 20,
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			out: "AUTHOR       EVENT            IP ADDRESS    DATE\n" +
				"developer    create.secret    127.0.0.1     2018-01-01T01:01:01+01:00\n",
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
				perPage: 20,
			},
			out: "AUTHOR    EVENT    IP ADDRESS    DATE\n",
		},
		"error secret version": {
			cmd: AuditCommand{
				path:    "namespace/repo/secret:1",
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
				perPage: 20,
			},
			err: testError,
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
				perPage: 20,
			},
			err: ErrInvalidAuditActor,
			out: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO()
			tc.cmd.io = io

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
