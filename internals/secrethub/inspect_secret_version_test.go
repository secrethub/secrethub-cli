package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"
	"time"

	"github.com/keylockerbv/secrethub-cli/internals/secrethub/fakes"
	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestInspectSecretVersion_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd                  InspectSecretVersionCommand
		secretVersionService fakeclient.SecretVersionService
		newClientErr         error
		out                  string
		err                  error
	}{
		"succes": {
			cmd: InspectSecretVersionCommand{
				path: "foo/bar/secret:latest",
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			secretVersionService: fakeclient.SecretVersionService{
				WithoutDataGetter: fakeclient.WithoutDataGetter{
					ArgPath: "foo/bar/secret:latest",
					ReturnsVersion: &api.SecretVersion{
						Version:   1,
						CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						Status:    api.StatusOK,
					},
				},
			},
			out: "" +
				"{\n" +
				"    \"Version\": 1,\n" +
				"    \"CreatedAt\": \"2018-01-01T01:01:01+01:00\",\n" +
				"    \"Status\": \"ok\"\n" +
				"}\n",
		},
		"client not fount": {
			newClientErr: testErr,
			err:          testErr,
		},
		"secret not found": {
			cmd: InspectSecretVersionCommand{
				path: "foo/bar/secret:latest",
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			secretVersionService: fakeclient.SecretVersionService{
				WithoutDataGetter: fakeclient.WithoutDataGetter{
					ArgPath:        "foo/bar/secret:latest",
					ReturnsVersion: nil,
					Err:            api.ErrSecretNotFound,
				},
			},
			out: "",
			err: api.ErrSecretNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						VersionService: &tc.secretVersionService,
					},
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}

}
