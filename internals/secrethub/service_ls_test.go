package secrethub

import (
	"errors"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestServiceLsCommand_Run(t *testing.T) {
	cases := map[string]struct {
		cmd            ServiceLsCommand
		serviceService fakeclient.ServiceService
		newClientErr   error
		out            string
		err            error
	}{
		"success": {
			cmd: ServiceLsCommand{
				serviceTable: keyServiceTable{},
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialType("key"),
							},
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
							Credential: &api.Credential{
								Type: api.CredentialType("key"),
							},
						},
					},
				},
			},
			out: "ID      DESCRIPTION  TYPE\ntest    foobar       key\nsecond  foobarbaz    key\n",
		},
		"success quiet": {
			cmd: ServiceLsCommand{
				quiet: true,
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
						},
					},
				},
			},
			out: "test\nsecond\n",
		},
		"success aws": {
			cmd: ServiceLsCommand{
				serviceTable: awsServiceTable{},
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeAWSSTS,
								Metadata: map[string]string{
									api.CredentialMetadataAWSRole:   "arn:aws:iam::123456:role/path/to/role",
									api.CredentialMetadataAWSKMSKey: "12345678-1234-1234-1234-123456789012",
								},
							},
						},
					},
				},
			},
			out: "ID    DESCRIPTION  ROLE                                   KMS-KEY\ntest  foobar       arn:aws:iam::123456:role/path/to/role  12345678-1234-1234-1234-123456789012\n",
		},
		"success aws filter": {
			cmd: ServiceLsCommand{
				serviceTable: awsServiceTable{},
				filters: []func(*api.Service) bool{
					isAWSService,
				},
			},
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					ReturnsServices: []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeAWSSTS,
								Metadata: map[string]string{
									api.CredentialMetadataAWSRole:   "arn:aws:iam::123456:role/path/to/role",
									api.CredentialMetadataAWSKMSKey: "arn:aws:kms:us-east-1:123456:key/12345678-1234-1234-1234-123456789012",
								},
							},
						},
						{
							ServiceID:   "test2",
							Description: "foobarbaz",
							Credential: &api.Credential{
								Type: api.CredentialTypeRSA,
							},
						},
					},
				},
			},
			out: "ID    DESCRIPTION  ROLE                                   KMS-KEY\ntest  foobar       arn:aws:iam::123456:role/path/to/role  arn:aws:kms:us-east-1:123456:key/12345678-1234-1234-1234-123456789012\n",
		},
		"new client error": {
			newClientErr: errors.New("error"),
			err:          errors.New("error"),
		},
		"client list error": {
			serviceService: fakeclient.ServiceService{
				Lister: fakeclient.RepoServiceLister{
					Err: errors.New("error"),
				},
			},
			err: errors.New("error"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.ClientAdapter, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.ClientAdapter, error) {
					return fakeclient.Client{
						ServiceService: &tc.serviceService,
					}, nil
				}
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
