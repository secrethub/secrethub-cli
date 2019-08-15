package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestServiceLsCommand_Run(t *testing.T) {
	cases := map[string]struct {
		cmd            ServiceLsCommand
		serviceService fakeclient.ServiceService
		newClientErr   error
		out            string
		err            error
	}{
		"success": {
			cmd: ServiceLsCommand{
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+00:00",
				},
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
						},
					},
				},
			},
			out: "ID      DESCRIPTION    CREATED\ntest    foobar         2018-01-01T01:01:01+00:00\nsecond  foobarbaz      2018-01-01T01:01:01+00:00\n",
		},
		"success quiet": {
			cmd: ServiceLsCommand{
				quiet: true,
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
						},
					},
				},
			},
			out: "test\nsecond\n",
		},
		"new client error": {
			newClientErr: errors.New("error"),
			err:          errors.New("error"),
		},
		"client list error": {
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					Err: errors.New("error"),
				},
			},
			err: errors.New("error"),
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
						ServiceService: &tc.serviceService,
					}, nil
				}
			}

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
