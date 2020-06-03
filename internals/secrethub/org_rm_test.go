package secrethub

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgRmCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgRmCommand
		deleteFunc   func(name string) error
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
			deleteFunc: func(name string) error {
				return testErr
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
			deleteFunc: func(name string) error {
				return nil
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
			var argName string

			// Setup
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					OrgService: &fakeclient.OrgService{
						DeleteFunc: func(name string) error {
							argName = name
							return tc.deleteFunc(name)
						}},
				}, tc.newClientErr
			}

			io := fakeui.NewIO(t)
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			io.PromptErr = tc.promptErr
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, argName, tc.argName)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
