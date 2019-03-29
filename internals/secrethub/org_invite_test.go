package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"bytes"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestOrgInviteCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	cases := map[string]struct {
		cmd          OrgInviteCommand
		service      fakeclient.OrgMemberService
		newClientErr error
		in           string
		promptOut    string
		out          string
		err          error
	}{
		"success": {
			cmd: OrgInviteCommand{
				orgName:  "company",
				username: "dev1",
				role:     api.OrgRoleMember,
			},
			service: fakeclient.OrgMemberService{
				Inviter: fakeclient.OrgInviter{
					ReturnsOrgMember: &api.OrgMember{
						User: &api.User{
							Username: "dev1",
						},
						Role: api.OrgRoleMember,
					},
				},
			},
			in:        "y",
			promptOut: "Are you sure you want to invite dev1 to the company organization? [y/N]: ",
			out: "Inviting user...\n" +
				"Invite complete! The user dev1 is now member of the company organization.\n",
		},
		"success force": {
			cmd: OrgInviteCommand{
				orgName:  "company",
				username: "dev1",
				role:     api.OrgRoleMember,
				force:    true,
			},
			service: fakeclient.OrgMemberService{
				Inviter: fakeclient.OrgInviter{
					ReturnsOrgMember: &api.OrgMember{
						User: &api.User{
							Username: "dev1",
						},
						Role: api.OrgRoleMember,
					},
				},
			},
			out: "Inviting user...\n" +
				"Invite complete! The user dev1 is now member of the company organization.\n",
		},
		"abort": {
			cmd: OrgInviteCommand{
				orgName:  "company",
				username: "dev1",
				role:     api.OrgRoleMember,
			},
			in:        "n",
			promptOut: "Are you sure you want to invite dev1 to the company organization? [y/N]: ",
			out:       "Aborting.\n",
		},
		"new client error": {
			cmd: OrgInviteCommand{
				orgName:  "company",
				username: "dev1",
				role:     api.OrgRoleMember,
				force:    true,
			},
			newClientErr: testErr,
			err:          testErr,
		},
		"invite error": {
			cmd: OrgInviteCommand{
				force: true,
			},
			service: fakeclient.OrgMemberService{
				Inviter: fakeclient.OrgInviter{
					Err: testErr,
				},
			},
			out: "Inviting user...\n",
			err: testErr,
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
			io.PromptIn.Buffer = bytes.NewBufferString(tc.in)
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
			assert.Equal(t, io.PromptOut.String(), tc.promptOut)
		})
	}
}
