package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestInspectSecret_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd                  InspectSecretCommand
		secretVersionService fakeclient.SecretVersionService
		newClientErr         error
		out                  string
		err                  error
	}{
		"succes one version": {
			cmd: InspectSecretCommand{
				path: "foo/bar/secret",
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			secretVersionService: fakeclient.SecretVersionService{
				GetWithoutDataFunc: func(path string) (*api.SecretVersion, error) {
					return &api.SecretVersion{
						Secret: &api.Secret{
							Name:         "secret",
							CreatedAt:    time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
							VersionCount: 1,
						},
						Version:   1,
						CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						Status:    api.StatusOK,
					}, nil
				},
				ListWithoutDataFunc: func(path string) ([]*api.SecretVersion, error) {
					return []*api.SecretVersion{
						{
							Secret: &api.Secret{
								Name:         "secret",
								CreatedAt:    time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
								VersionCount: 1,
							},
							Version:   1,
							CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
							Status:    api.StatusOK,
						},
					}, nil
				},
			},
			out: "" +
				"{\n" +
				"    \"Name\": \"secret\",\n" +
				"    \"CreatedAt\": \"2018-01-01T01:01:01+01:00\",\n" +
				"    \"VersionCount\": 1,\n" +
				"    \"Versions\": [\n" +
				"        {\n" +
				"            \"Version\": 1,\n" +
				"            \"CreatedAt\": \"2018-01-01T01:01:01+01:00\",\n" +
				"            \"Status\": \"ok\"\n" +
				"        }\n" +
				"    ]\n" +
				"}\n",
		},
		"no secret": {
			cmd: InspectSecretCommand{
				path: "foo/bar/secret",
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-01-01T01:01:01+01:00",
				},
			},
			secretVersionService: fakeclient.SecretVersionService{
				GetWithoutDataFunc: func(path string) (*api.SecretVersion, error) {
					return nil, api.ErrSecretNotFound
				},
			},
			out: "",
			err: api.ErrSecretNotFound,
		},
		"no client": {
			newClientErr: testErr,
			err:          testErr,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Arrange
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					SecretService: &fakeclient.SecretService{
						VersionService: &tc.secretVersionService,
					},
				}, tc.newClientErr
			}

			io := fakeui.NewIO(t)
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}

}
