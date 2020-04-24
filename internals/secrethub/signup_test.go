package secrethub

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/progress/fakeprogress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestSignUpCommand_Run(t *testing.T) {
	cases := map[string]struct {
		cmd       SignUpCommand
		promptIn  string
		promptOut string
		out       string
		err       error
	}{
		// TODO SHDEV-1029: Test.
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO()
			tc.cmd.io = io
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)

			progressPrinter := fakeprogress.Printer{}
			tc.cmd.progressPrinter = &progressPrinter
			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					UserService: &fakeclient.UserService{},
				}, nil
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
