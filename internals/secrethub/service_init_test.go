package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-go/internals/errio"
	"os"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

//type credentialMock struct {
//	Key []byte
//}
//
//func (c *credentialMock) Verifier() credentials.Verifier {
//	panic("implement me")
//}
//
//func (c *credentialMock) Encrypter() credentials.Encrypter {
//	return nil
//}
//
//func (c *credentialMock) Metadata() map[string]string {
//	return nil
//}
//
//func (c *credentialMock) Create() error {
//	c.Key = []byte("CredentialTest")
//	return nil
//}
//
//func (c *credentialMock) Export() ([]byte, error) {
//	return c.Key, nil
//}

func TestServiceInitCommand_Run(t *testing.T) {
	keyCreator := credentials.CreateKey()
	keyCreator.Create()
	val, _ := keyCreator.Export()
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd             ServiceInitCommand
		serviceService  fakeclient.ServiceService
		setFunc         func(path string, permission string, accountName string) (*api.AccessRule, error)
		expectedPerm    *api.AccessRule
		newClientErr    error
		expectedFileOut []byte
		expectedOut     string
		fileErr         error
		err             error
	}{
		"success": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{}, nil
				},
			},
			expectedOut: string(val) + "\n",
		},
		"write to file": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
				file:       "test.txt",
				fileMode:   filemode.New(os.ModePerm),
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			expectedFileOut: []byte(string(val) + "\n"),
			expectedOut:     "Written account configuration for testService to test.txt. Be sure to remove it when you're done.\n",
		},
		"fail write to file": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			err: ErrCannotWrite("path/test.txt", "open path/test.txt: no such file or directory"),
		},
		"give 1 permission": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			setFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
				return &api.AccessRule{
					Permission: api.PermissionRead,
				}, nil
			},
			expectedPerm: &api.AccessRule{Permission: api.PermissionRead},
			expectedOut:  string(val) + "\n",
		},
		"give 2 permissions": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
				permission: "read:write",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
			},
			setFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
				if permission == "read" {
					return &api.AccessRule{
						Permission: api.PermissionRead,
					}, nil
				} else if permission == "write" {
					return &api.AccessRule{
						Permission: api.PermissionWrite,
					}, nil
				}
				return nil, testErr
			},
			expectedPerm: &api.AccessRule{Permission: api.PermissionWrite},
			expectedOut:  string(val) + "\n",
		},
		"fail permission": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
				DeleteFunc: func(id string) (*api.RevokeRepoResponse, error) {
					return &api.RevokeRepoResponse{}, nil
				},
			},
			setFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
				return &api.AccessRule{}, testErr
			},
			err: testErr,
		},
		"fail permission and revoke": {
			cmd: ServiceInitCommand{
				repo:       api.RepoPath("test/repo"),
				credential: keyCreator,
				permission: "read",
			},
			serviceService: fakeclient.ServiceService{
				CreateFunc: func(path string, description string, credentialCreator credentials.Creator) (*api.Service, error) {
					return &api.Service{
						ServiceID: "testService",
					}, nil
				},
				DeleteFunc: func(id string) (*api.RevokeRepoResponse, error) {
					return nil, fmt.Errorf("revoke has failed")
				},
			},
			setFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
				return &api.AccessRule{}, testErr
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
			var perm *api.AccessRule
			var fileOut []byte
			testIO := fakeui.NewIO(t)
			tc.cmd.io = testIO

			tc.cmd.newClient = func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					ServiceService: &tc.serviceService,
					AccessRuleService: &fakeclient.AccessRuleService{
						SetFunc: func(path string, permission string, accountName string) (*api.AccessRule, error) {
							returnedPerm, err := tc.setFunc(path, permission, accountName)
							perm = returnedPerm
							return returnedPerm, err
						},
					},
				}, tc.err
			}
			tc.cmd.writeFileFunc = func(filename string, data []byte, perm os.FileMode) error {
				if tc.fileErr == nil {
					fileOut = data
				}
				return tc.fileErr
			}

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, perm, tc.expectedPerm)
			assert.Equal(t, testIO.Out.String(), tc.expectedOut)
			assert.Equal(t, fileOut, tc.expectedFileOut)
		})
	}
}
