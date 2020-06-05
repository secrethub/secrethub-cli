package secrethub

import (
	"errors"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

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
				newServiceTable: newKeyServiceTable,
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialType("key"),
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
							Credential: &api.Credential{
								Type: api.CredentialType("key"),
							},
							CreatedAt: time.Now().Add(-2 * time.Hour),
						},
					}, nil
				},
			},
			out: "" +
				"ID      DESCRIPTION  TYPE  CREATED\n" +
				"test    foobar       key   About an hour ago\nsecond  foobarbaz    key   2 hours ago\n",
		},
		"success quiet": {
			cmd: ServiceLsCommand{
				quiet: true,
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
						},
						{
							ServiceID:   "second",
							Description: "foobarbaz",
						},
					}, nil
				},
			},
			out: "test\nsecond\n",
		},
		"success aws": {
			cmd: ServiceLsCommand{
				newServiceTable: newAWSServiceTable,
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeAWS,
								Metadata: map[string]string{
									api.CredentialMetadataAWSRole:   "arn:aws:iam::123456:role/path/to/role",
									api.CredentialMetadataAWSKMSKey: "12345678-1234-1234-1234-123456789012",
								},
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
					}, nil
				},
			},
			out: "" +
				"ID    DESCRIPTION  ROLE                                   KMS-KEY                               CREATED\n" +
				"test  foobar       arn:aws:iam::123456:role/path/to/role  12345678-1234-1234-1234-123456789012  About an hour ago\n",
		},
		"success aws filter": {
			cmd: ServiceLsCommand{
				newServiceTable: newAWSServiceTable,
				filters: []func(*api.Service) bool{
					isAWSService,
				},
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeAWS,
								Metadata: map[string]string{
									api.CredentialMetadataAWSRole:   "arn:aws:iam::123456:role/path/to/role",
									api.CredentialMetadataAWSKMSKey: "arn:aws:kms:us-east-1:123456:key/12345678-1234-1234-1234-123456789012",
								},
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
						{
							ServiceID:   "test2",
							Description: "foobarbaz",
							Credential: &api.Credential{
								Type: api.CredentialTypeKey,
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
					}, nil
				},
			},
			out: "" +
				"ID    DESCRIPTION  ROLE                                   KMS-KEY                                                                CREATED\n" +
				"test  foobar       arn:aws:iam::123456:role/path/to/role  arn:aws:kms:us-east-1:123456:key/12345678-1234-1234-1234-123456789012  About an hour ago\n",
		},
		"success gcp": {
			cmd: ServiceLsCommand{
				newServiceTable: newGCPServiceTable,
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeGCPServiceAccount,
								Metadata: map[string]string{
									api.CredentialMetadataGCPServiceAccountEmail: "service-account@secrethub-test-1234567890.iam.gserviceaccount.com",
									api.CredentialMetadataGCPKMSKeyResourceID:    "projects/secrethub-test-1234567890.iam/locations/global/keyRings/test/cryptoKeys/test",
								},
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
					}, nil
				},
			},
			out: "" +
				"ID    DESCRIPTION  SERVICE-ACCOUNT-EMAIL                                              KMS-KEY                                                                                CREATED\n" +
				"test  foobar       service-account@secrethub-test-1234567890.iam.gserviceaccount.com  projects/secrethub-test-1234567890.iam/locations/global/keyRings/test/cryptoKeys/test  About an hour ago\n",
		},
		"success gcp filter": {
			cmd: ServiceLsCommand{
				newServiceTable: newGCPServiceTable,
				filters: []func(*api.Service) bool{
					isGCPService,
				},
			},
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return []*api.Service{
						{
							ServiceID:   "test",
							Description: "foobar",
							Credential: &api.Credential{
								Type: api.CredentialTypeGCPServiceAccount,
								Metadata: map[string]string{
									api.CredentialMetadataGCPServiceAccountEmail: "service-account@secrethub-test-1234567890.iam.gserviceaccount.com",
									api.CredentialMetadataGCPKMSKeyResourceID:    "projects/secrethub-test-1234567890.iam/locations/global/keyRings/test/cryptoKeys/test",
								},
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
						{
							ServiceID:   "test2",
							Description: "foobarbaz",
							Credential: &api.Credential{
								Type: api.CredentialTypeKey,
							},
							CreatedAt: time.Now().Add(-1 * time.Hour),
						},
					}, nil
				},
			},
			out: "" +
				"ID    DESCRIPTION  SERVICE-ACCOUNT-EMAIL                                              KMS-KEY                                                                                CREATED\n" +
				"test  foobar       service-account@secrethub-test-1234567890.iam.gserviceaccount.com  projects/secrethub-test-1234567890.iam/locations/global/keyRings/test/cryptoKeys/test  About an hour ago\n",
		},
		"new client error": {
			newClientErr: errors.New("error"),
			err:          errors.New("error"),
		},
		"client list error": {
			serviceService: fakeclient.ServiceService{
				ListFunc: func(path string) ([]*api.Service, error) {
					return nil, errors.New("error")
				},
			},
			err: errors.New("error"),
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

			// Act
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
