package secrethub

import (
	"errors"
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	faketimeformatter "github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLListCommand_run(t *testing.T) {
	testError := errio.Error(errors.New("test error"))

	dir1ID := uuid.New()
	dir2ID := uuid.New()

	cases := map[string]struct {
		cmd          ACLListCommand
		newClientErr error
		accessrules  fakeclient.AccessRuleService
		dirs         fakeclient.DirService
		argPath      api.DirPath
		argDepth     int
		argAncestors bool
		out          string
		err          error
	}{
		"client creation error": {
			cmd:          ACLListCommand{},
			newClientErr: testError,
			err:          testError,
		},
		"client error": {
			cmd: ACLListCommand{},
			accessrules: fakeclient.AccessRuleService{
				Lister: &fakeclient.AccessRuleLister{
					Err: testError,
				},
			},
			err: testError,
		},
		"0 access rules": {
			cmd: ACLListCommand{},
			accessrules: fakeclient.AccessRuleService{
				Lister: &fakeclient.AccessRuleLister{
					ReturnsAccessRules: []*api.AccessRule{},
				},
			},
			out: "PATH    PERMISSIONS    LAST EDITED    ACCOUNT\n",
		},
		"args": {
			cmd: ACLListCommand{
				path:      api.DirPath("namespace/repo/dir"),
				depth:     1,
				ancestors: true,
			},
			accessrules: fakeclient.AccessRuleService{
				Lister: &fakeclient.AccessRuleLister{
					ReturnsAccessRules: []*api.AccessRule{},
				},
			},
			argPath:      api.DirPath("namespace/repo/dir"),
			argDepth:     1,
			argAncestors: true,
			out:          "PATH    PERMISSIONS    LAST EDITED    ACCOUNT\n",
		},
		"success": {
			cmd: ACLListCommand{
				timeFormatter: &faketimeformatter.TimeFormatter{
					Response: "1 hour ago",
				},
			},
			accessrules: fakeclient.AccessRuleService{
				Lister: &fakeclient.AccessRuleLister{
					ReturnsAccessRules: []*api.AccessRule{
						{
							Account: &api.Account{
								Name: "another dev",
							},
							DirID:         dir1ID,
							Permission:    api.PermissionWrite,
							LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
						{
							Account: &api.Account{
								Name: "developer",
							},
							DirID:         dir1ID,
							Permission:    api.PermissionRead,
							LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
						{
							Account: &api.Account{
								Name: "developer",
							},
							DirID:         dir2ID,
							Permission:    api.PermissionAdmin,
							LastChangedAt: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
						},
					},
				},
			},
			dirs: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					ReturnsTree: &api.Tree{
						ParentPath: "namespace",
						Dirs: map[uuid.UUID]*api.Dir{
							*dir1ID: {
								Name:  "repo",
								DirID: dir1ID,
							},
							*dir2ID: {
								Name:     "dir",
								DirID:    dir2ID,
								ParentID: dir1ID,
							},
						},
						RootDir: &api.Dir{
							Name:  "repo",
							DirID: dir1ID,
						},
					},
				},
			},
			out: "PATH                  PERMISSIONS    LAST EDITED    ACCOUNT\n" +
				"namespace/repo        write          1 hour ago     another dev\n" +
				"namespace/repo        read           1 hour ago     developer\n" +
				"namespace/repo/dir    admin          1 hour ago     developer\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					AccessRuleService: &tc.accessrules,
					DirService:        &tc.dirs,
				}, tc.newClientErr
			}

			// Act
			err := tc.cmd.run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
