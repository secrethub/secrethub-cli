package secrethub

import (
	"testing"

	"bytes"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/progress/fakeprogress"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestSignUpCommand_Run(t *testing.T) {
	testutil.Unit(t)

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
			io := ui.NewFakeIO()
			tc.cmd.io = io
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)

			progressPrinter := fakeprogress.Printer{}
			tc.cmd.progressPrinter = &progressPrinter
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					UserService: &fakeclient.UserService{},
				}, nil
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.PromptOut.String(), tc.promptOut)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
