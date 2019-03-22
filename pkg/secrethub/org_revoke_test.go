package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"bytes"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgRevokeCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgRevokeCommand
		promptIn     string
		service      fakeclient.OrgMemberService
		newClientErr error
		out          string
		err          error
	}{
		"success, not a repo member": {
			cmd: OrgRevokeCommand{
				orgName:  "company",
				username: "dev1",
			},
			promptIn: "dev1",
			service: fakeclient.OrgMemberService{
				Revoker: fakeclient.OrgMemberRevoker{
					ReturnsRevokeOrgResponse: &api.RevokeOrgResponse{
						Repos: []*api.RevokeRepoResponse{},
					},
				},
			},
			out: "The user dev1 has no memberships to any of company's repos and can be safely removed.\n" +
				"\n" +
				"\n" +
				"Revoking user...\n" +
				"Revoke complete!\n",
		},
		"success, repo member": {
			cmd: OrgRevokeCommand{
				orgName:  "company",
				username: "dev1",
			},
			promptIn: "dev1",
			service: fakeclient.OrgMemberService{
				Revoker: fakeclient.OrgMemberRevoker{
					ReturnsRevokeOrgResponse: &api.RevokeOrgResponse{
						Repos: []*api.RevokeRepoResponse{
							{
								Namespace: "company",
								Name:      "application1",
								Status:    api.StatusOK,
							},
							{
								Namespace: "company",
								Name:      "application2",
								Status:    api.StatusFlagged,
							},
							{
								Namespace: "company",
								Name:      "application3",
								Status:    api.StatusFailed,
							},
						},
						StatusCounts: map[string]int{
							api.StatusOK:      1,
							api.StatusFlagged: 1,
							api.StatusFailed:  1,
						},
					},
				},
			},
			out: "[WARNING] Revoking dev1 from the company organization will revoke the user from 3 repositories, automatically flagging secrets for rotation.\n" +
				"\n" +
				"A revocation plan has been generated and is shown below. Flagged repositories will contain secrets flagged for rotation, failed repositories require a manual removal or access rule changes before proceeding and OK repos will not require rotation.\n" +
				"\n" +
				"  company/application1  => ok\n" +
				"  company/application2  => flagged\n" +
				"  company/application3  => failed\n" +
				"\n" +
				"Revocation plan: 1 to flag, 1 to fail, 1 OK.\n" +
				"\n" +
				"\n" +
				"Revoking user...\n" +
				"\n" +
				"  company/application1  => ok\n" +
				"  company/application2  => flagged\n" +
				"  company/application3  => failed\n" +
				"\n" +
				"Revoke complete! Repositories: 1 flagged, 1 failed, 1 OK.\n",
		},
		"abort": {
			cmd: OrgRevokeCommand{
				orgName:  "company",
				username: "dev1",
			},
			promptIn: "typo",
			service: fakeclient.OrgMemberService{
				Revoker: fakeclient.OrgMemberRevoker{
					ReturnsRevokeOrgResponse: &api.RevokeOrgResponse{
						Repos: []*api.RevokeRepoResponse{},
					},
				},
			},
			out: "The user dev1 has no memberships to any of company's repos and can be safely removed.\n" +
				"\n" +
				"Name does not match. Aborting.\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					OrgService: &fakeclient.OrgService{
						MemberService: &tc.service,
					},
				}, tc.newClientErr
			}

			io := ui.NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
