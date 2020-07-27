package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"os"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestServiceInitCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")
	//var credential credentials.Creator

	cases := map[string]struct {
		cmd            ServiceInitCommand
		serviceService fakeclient.ServiceService
		newClientErr   error
		out            string
		err            error
	}{
		//"success service init": {
		//	cmd: ServiceInitCommand{
		//		repo: api.RepoPath("test/repo"),
		//	},
		//	serviceService: fakeclient.ServiceService{
		//		CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
		//			//credential.Create()
		//			//credentialCreator = &credential
		//			credentialCreator.Create()
		//			credential = credentialCreator
		//
		//			return &api.Service{}, nil
		//		},
		//	},
		//	out: "",
		//},
		"fail no key created": {
			cmd: ServiceInitCommand{
				repo: api.RepoPath("test/repo"),
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{}, nil
				},
			},
			err: fmt.Errorf("key has not yet been generated created. Use KeyCreator before calling Export()"),
		},
		"init fail file exists": {
			cmd: ServiceInitCommand{
				file: "test.txt",
			},
			err: ErrFileAlreadyExists,
		},
		"init fail flags conflict": {
			cmd: ServiceInitCommand{
				clip: true,
				file: "test.txt",
			},
			err: ErrFlagsConflict("--clip and --file"),
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"service init error": {
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return nil, testErr
				},
			},
			err: testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO(t)
			tc.cmd.io = io

			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						ServiceService: &tc.serviceService,
					}, nil
				}
			}

			if name == "init fail file exists" {
				os.Create("test.txt")
				defer os.Remove("test.txt")
			}

			// Run
			err := tc.cmd.Run()
			//out, _, _ := credential.Verifier().Export()
			//tc.out = string(out)

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
