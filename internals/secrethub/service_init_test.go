package secrethub

import (
	"fmt"
	"os"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestServiceInitCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd               ServiceInitCommand
		serviceService    fakeclient.ServiceService
		accessRuleService fakeclient.AccessRuleService
		newClientErr      error
		out               string
		err               error
	}{
		"success service init": {
			cmd: ServiceInitCommand{
				repo: api.RepoPath("test/repo"),
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{}, nil
				},
			},
			out: "",
		},
		"success write to file": {
			cmd: ServiceInitCommand{
				repo:     api.RepoPath("test/repo"),
				file:     "test.txt",
				fileMode: filemode.New(os.ModePerm),
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			out: "Written account configuration for testService to test.txt. Be sure to remove it when you're done.\n",
		},
		"fail write to file": {
			cmd: ServiceInitCommand{
				repo:     api.RepoPath("test/repo"),
				file:     "path/test.txt",
				fileMode: filemode.New(os.ModePerm),
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			err: ErrCannotWrite("path/test.txt", "open path/test.txt: no such file or directory"),
		},
		// TODO: Check permission on the created service account
		"give 1 permission": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				file:       "test.txt",
				fileMode:   filemode.New(os.ModePerm),
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			accessRuleService: fakeclient.AccessRuleService{
				SetFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
					return &api.AccessRule{
						Permission: api.PermissionRead,
					}, nil
				},
			},
			out: "Written account configuration for testService to test.txt. Be sure to remove it when you're done.\n",
		},
		"give 2 permissions": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				file:       "test.txt",
				fileMode:   filemode.New(os.ModePerm),
				permission: "read:write",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			accessRuleService: fakeclient.AccessRuleService{
				SetFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
					if permission == "read" {
						return &api.AccessRule{
							Permission: api.PermissionRead,
						}, nil
					} else if permission == "write" {
						return &api.AccessRule{
							Permission: api.PermissionRead,
						}, nil
					}
					return nil, testErr
				},
			},
			out: "Written account configuration for testService to test.txt. Be sure to remove it when you're done.\n",
		},
		"fail permission": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				file:       "test.txt",
				fileMode:   filemode.New(os.ModePerm),
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
				DeleteFunc: func(id string) (*api.RevokeRepoResponse, error) {
					return &api.RevokeRepoResponse{}, nil
				},
			},
			accessRuleService: fakeclient.AccessRuleService{
				SetFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
					return &api.AccessRule{}, testErr
				},
			},
			err: testErr,
		},
		"fail permission and revoke": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				file:       "test.txt",
				fileMode:   filemode.New(os.ModePerm),
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					_ = credentialCreator.Create()
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
				DeleteFunc: func(id string) (*api.RevokeRepoResponse, error) {
					return nil, fmt.Errorf("revoke has failed")
				},
			},
			accessRuleService: fakeclient.AccessRuleService{
				SetFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
					return &api.AccessRule{}, testErr
				},
			},
			err: fmt.Errorf("revoke has failed"),
		},
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
						ServiceService:    &tc.serviceService,
						AccessRuleService: &tc.accessRuleService,
					}, nil
				}
			}

			if name == "init fail file exists" {
				_, _ = os.Create("test.txt")
			}

			// Run
			err := tc.cmd.Run()
			if name == "success service init" {
				//TODO Remove this condition when service credential can be checked
				io = fakeui.NewIO(t)
			}
			if _, err := os.Stat("test.txt"); err == nil {
				defer os.Remove("test.txt")
			}

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
