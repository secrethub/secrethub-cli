package secrethub

import (
	"testing"

	"bytes"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgRmCommand_Run(t *testing.T) {
	testutil.Unit(t)

	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgRmCommand
		service      fakeclient.OrgService
		newClientErr error
		promptIn     string
		promptOut    string
		promptErr    error
		argName      string
		err          error
		out          string
	}{
		"client creation error": {
			cmd: OrgRmCommand{
				name: "organization",
			},
			newClientErr: testErr,
			promptIn:     "organization",
			out:          "",
			promptOut:    "[DANGER ZONE] This action cannot be undone. This will permanently delete the organization organization, repositories, and remove all team associations. Please type in the name of the organization to confirm: ",
			err:          testErr,
		},
		"client error": {
			cmd: OrgRmCommand{
				name: "organization",
			},
			service: fakeclient.OrgService{
				Deleter: fakeclient.OrgDeleter{
					Err: testErr,
				},
			},
			promptIn:  "organization",
			argName:   "organization",
			err:       testErr,
			promptOut: "[DANGER ZONE] This action cannot be undone. This will permanently delete the organization organization, repositories, and remove all team associations. Please type in the name of the organization to confirm: ",
			out:       "Deleting organization...\n",
		},
		"abort": {
			cmd: OrgRmCommand{
				name: "organization",
			},
			promptIn:  "",
			promptOut: "[DANGER ZONE] This action cannot be undone. This will permanently delete the organization organization, repositories, and remove all team associations. Please type in the name of the organization to confirm: ",
			out:       "Name does not match. Aborting.\n",
		},
		"success": {
			cmd: OrgRmCommand{
				name: "organization",
			},
			service: fakeclient.OrgService{
				Deleter: fakeclient.OrgDeleter{},
			},
			promptIn:  "organization",
			argName:   "organization",
			promptOut: "[DANGER ZONE] This action cannot be undone. This will permanently delete the organization organization, repositories, and remove all team associations. Please type in the name of the organization to confirm: ",
			out: "Deleting organization...\n" +
				"Delete complete! The organization organization has been permanently deleted.\n",
		},
		"confirm error": {
			cmd: OrgRmCommand{
				name: "organization",
			},
			promptErr: testErr,
			err:       testErr,
			out:       "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					OrgService: &tc.service,
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.PromptOut.String(), tc.promptOut)
			testutil.Compare(t, tc.service.Deleter.ArgName, tc.argName)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
