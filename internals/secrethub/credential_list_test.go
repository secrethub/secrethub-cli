package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"
)

type credentialIterator struct {
	credentials  []*api.Credential
	currentIndex int
	err          error
}

func (c *credentialIterator) Next() (api.Credential, error) {
	if c.err != nil {
		return api.Credential{}, c.err
	}

	currentIndex := c.currentIndex
	if currentIndex >= len(c.credentials) {
		return api.Credential{}, iterator.Done
	}
	c.currentIndex++
	return *c.credentials[currentIndex], nil
}

func TestCredentialListCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd               CredentialListCommand
		credentialService fakeclient.CredentialService
		newClientErr      error
		out               string
		err               error
	}{
		"success list credentials": {
			cmd: CredentialListCommand{},
			credentialService: fakeclient.CredentialService{
				ListFunc: func(_ *secrethub.CredentialListParams) secrethub.CredentialIterator {
					return &credentialIterator{
						credentials: []*api.Credential{
							{
								Description: "credential 1",
								Type:        "test",
								Fingerprint: "8E146D837D4CA1DC4315167B11A39C92",
							},
							{
								Description: "credential 2",
								Enabled:     true,
								Fingerprint: "DFC3D1F0D9842F17425403D0A9474C36",
							},
						},
						currentIndex: 0,
						err:          nil,
					}
				},
			},
			out: "FINGERPRINT       TYPE  ENABLED  CREATED        DESCRIPTION\n" +
				"8E146D837D4CA1DC  test  no       292 years ago  credential 1\n" +
				"DFC3D1F0D9842F17        yes      292 years ago  credential 2\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
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
						CredentialService: &tc.credentialService,
					}, nil
				}
			}

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.Out.String(), tc.out)
		})
	}
}
