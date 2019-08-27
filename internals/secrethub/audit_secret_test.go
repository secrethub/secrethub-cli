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

func TestAuditSecretCommand_run(t *testing.T) {
	testError := errors.New("test error")

	cases := map[string]struct {
		cmd AuditSecretCommand
		err error
		out string
	}{
		"create secret event": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								ReturnsAuditEvents: []*api.Audit{
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
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			out: "AUTHOR       EVENT            IP ADDRESS    DATE\n" +
				"developer    create.secret    127.0.0.1     2018-01-01T01:01:01+01:00\n",
		},
		"0 events": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								ReturnsAuditEvents: []*api.Audit{},
							},
						},
					}, nil
				},
			},
			out: "AUTHOR    EVENT    IP ADDRESS    DATE\n",
		},
		"error secret version": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/secret:1",
			},
			err: ErrCannotAuditSecretVersion,
		},
		"client creation error": {
			cmd: AuditSecretCommand{
				newClient: func() (secrethub.ClientAdapter, error) {
					return nil, ErrCannotFindHomeDir()
				},
			},
			err: ErrCannotFindHomeDir(),
		},
		"error can not audit dir": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/dir",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								Err: ErrCannotAuditDir,
							},
						},
					}, nil
				},
			},
			err: ErrCannotAuditDir,
		},
		"error secret not found": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						DirService: &fakeclient.DirService{
							TreeGetter: fakeclient.TreeGetter{
								Err: api.ErrDirNotFound,
							},
						},
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								Err: api.ErrSecretNotFound,
							},
						},
					}, nil
				},
			},
			err: api.ErrSecretNotFound,
		},
		"other list audit events error": {
			cmd: AuditSecretCommand{
				path: "namespace/repo/secret",
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								Err: testError,
							},
						},
					}, nil
				},
			},
			err: testError,
		},
		"invalid audit actor": {
			cmd: AuditSecretCommand{
				newClient: func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							EventLister: fakeclient.SecretEventLister{
								ReturnsAuditEvents: []*api.Audit{
									{},
								},
							},
						},
					}, nil
				},
			},
			err: ErrInvalidAuditActor,
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
