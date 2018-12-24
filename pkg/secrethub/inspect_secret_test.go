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

func TestInspectSecret_Run(t *testing.T) {
	testutil.Unit(t)

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
				WithoutDataGetter: fakeclient.WithoutDataGetter{
					ArgPath: "foo/bar/secret",
					ReturnsVersion: &api.SecretVersion{
						Secret: &api.Secret{
							Name:         "secret",
							CreatedAt:    time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
							VersionCount: 1,
						},
						Version:   1,
						CreatedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						Status:    api.StatusOK,
					},
				},
				WithoutDataLister: fakeclient.WithoutDataLister{
					ArgPath: "foo/bar/secret:1",
					ReturnsVersions: []*api.SecretVersion{
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
					},
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
				WithoutDataGetter: fakeclient.WithoutDataGetter{
					ArgPath:        "foo/bar/secret",
					ReturnsVersion: nil,
					Err:            api.ErrSecretNotFound,
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
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}

}
