package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgInitCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgInitCommand
		service      fakeclient.OrgService
		newClientErr error
		out          string
		err          error
	}{
		"success": {
			cmd: OrgInitCommand{
				name:        "company",
				description: "description",
				force:       true,
			},
			service: fakeclient.OrgService{
				CreateFunc: func(name string, description string) (*api.Org, error) {
					return &api.Org{
						Name: "company",
					}, nil
				},
			},
			out: "Creating organization...\n" +
				"Creation complete! The organization company is now ready to use.\n",
		},
		"new client error": {
			cmd: OrgInitCommand{
				name:        "company",
				description: "description",
				force:       true,
			},
			newClientErr: testErr,
			err:          testErr,
		},
		"create org error": {
			cmd: OrgInitCommand{
				name:        "company",
				description: "description",
				force:       true,
			},
			service: fakeclient.OrgService{
					CreateFunc: func(name string, description string) (*api.Org, error) {
						return nil, testErr
					},
			},
			out: "Creating organization...\n",
			err: testErr,
		},
	}
	// TODO SHDEV-1029: Test asking for missing args after these are refactored.

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					OrgService: &tc.service,
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
