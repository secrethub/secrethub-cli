package secrethub

import (
	"github.com/secrethub/secrethub-go/internals/assert"
	"testing"

	"github.com/keylockerbv/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestRepoRevokeCommand_Run(t *testing.T) {
	testErr := errio.Namespace("test").Code("test").Error("test error")

	testUUID := uuid.New()

	cases := map[string]struct {
		cmd            RepoRevokeCommand
		accountService fakeclient.AccountService
		dirService     fakeclient.DirService
		repoService    fakeclient.RepoService
		userService    fakeclient.UserService
		serviceService fakeclient.ServiceService
		newClientErr   error
		out            string
		err            error
	}{
		"revoke service force success, no flagged secrets": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("s-hTvStO9KaswJ"),
				path:        "namespace/repo",
				force:       true,
			},
			accountService: fakeclient.AccountService{
				Getter: fakeclient.AccountGetter{
					ReturnsAccount: &api.Account{
						AccountID: testUUID,
					},
				},
			},
			dirService: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					ReturnsTree: &api.Tree{
						RootDir: &api.Dir{
							Name: "repo",
						},
					},
				},
			},
			serviceService: fakeclient.ServiceService{
				Deleter: fakeclient.ServiceDeleter{
					ReturnsRevokeResponse: &api.RevokeRepoResponse{
						Status: api.StatusOK,
					},
				},
			},
			out: "Revoking account...\n\n" +
				"Revoke complete! The account s-hTvStO9KaswJ can no longer access the namespace/repo repository. Make sure you overwrite or delete all flagged secrets. Secrets: 0 unaffected, 0 flagged\n",
		},
		"revoke user force success, no flagged secrets": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
				path:        "namespace/repo",
				force:       true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						AccountID: testUUID,
						Username:  "dev1",
						FullName:  "Developer Uno",
					},
				},
			},
			dirService: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					ReturnsTree: &api.Tree{
						RootDir: &api.Dir{
							Name: "repo",
						},
					},
				},
			},
			repoService: fakeclient.RepoService{
				UserService: &fakeclient.RepoUserService{
					Revoker: fakeclient.RepoRevoker{
						ReturnsRevokeResponse: &api.RevokeRepoResponse{
							Status: api.StatusOK,
						},
					},
				},
			},
			out: "Revoking account...\n\n" +
				"Revoke complete! The account dev1 (Developer Uno) can no longer access the namespace/repo repository. Make sure you overwrite or delete all flagged secrets. Secrets: 0 unaffected, 0 flagged\n",
		},
		"revoke user force success, flagged secrets": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
				path:        "namespace/repo",
				force:       true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						AccountID: testUUID,
						Username:  "dev1",
						FullName:  "Developer Uno",
					},
				},
			},
			dirService: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					ReturnsTree: &api.Tree{
						RootDir: &api.Dir{
							Name:   "repo",
							Status: api.StatusFlagged,
							SubDirs: []*api.Dir{
								{
									Name: "dir",
									SubDirs: []*api.Dir{
										{
											Name: "subdir",
											Secrets: []*api.Secret{
												{
													Name:   "subsecret",
													Status: api.StatusOK,
												},
												{
													Name:   "flaggedsubsecret",
													Status: api.StatusFlagged,
												},
											},
											Status: api.StatusFlagged,
										},
									},
									Secrets: []*api.Secret{
										{
											Name:   "secret",
											Status: api.StatusOK,
										},
										{
											Name:   "flaggedsecret",
											Status: api.StatusFlagged,
										},
									},
									Status: api.StatusFlagged,
								},
							},
							Secrets: []*api.Secret{
								{
									Name:   "root secret",
									Status: api.StatusOK,
								},
								{
									Name:   "flagged root secret",
									Status: api.StatusFlagged,
								},
							},
						},
						Dirs: map[uuid.UUID]*api.Dir{},
					},
				},
			},
			repoService: fakeclient.RepoService{
				UserService: &fakeclient.RepoUserService{
					Revoker: fakeclient.RepoRevoker{
						ReturnsRevokeResponse: &api.RevokeRepoResponse{
							Status: api.StatusOK,
						},
					},
				},
			},
			out: "Revoking account...\n\n" +
				"namespace/repo/dir/subdir/flaggedsubsecret  => flagged\n" +
				"namespace/repo/dir/flaggedsecret            => flagged\n" +
				"namespace/repo/flagged root secret          => flagged\n" +
				"\n" +
				"Revoke complete! The account dev1 (Developer Uno) can no longer access the namespace/repo repository. Make sure you overwrite or delete all flagged secrets. Secrets: 3 unaffected, 3 flagged\n",
		},
		"new client error": {
			newClientErr: testErr,
			err:          testErr,
		},
		"get user error": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					Err: testErr,
				},
			},
			err: testErr,
		},
		"service delete error": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("s-hTvStO9KaswJ"),
				path:        "namespace/repo",
				force:       true,
			},
			accountService: fakeclient.AccountService{
				Getter: fakeclient.AccountGetter{
					ReturnsAccount: &api.Account{
						AccountID: testUUID,
					},
				},
			},
			serviceService: fakeclient.ServiceService{
				Deleter: fakeclient.ServiceDeleter{
					Err: testErr,
				},
			},
			out: "Revoking account...\n\n",
			err: testErr,
		},
		"user revoke error": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
				path:        "namespace/repo",
				force:       true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						AccountID: testUUID,
						Username:  "dev1",

						FullName: "Developer Uno",
					},
				},
			},
			repoService: fakeclient.RepoService{
				UserService: &fakeclient.RepoUserService{
					Revoker: fakeclient.RepoRevoker{
						Err: testErr,
					},
				},
			},
			out: "Revoking account...\n\n",
			err: testErr,
		},
		"get tree error": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
				path:        "namespace/repo",
				force:       true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						AccountID: testUUID,
						Username:  "dev1",

						FullName: "Developer Uno",
					},
				},
			},
			dirService: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					Err: testErr,
				},
			},
			repoService: fakeclient.RepoService{
				UserService: &fakeclient.RepoUserService{
					Revoker: fakeclient.RepoRevoker{
						ReturnsRevokeResponse: &api.RevokeRepoResponse{
							Status: api.StatusOK,
						},
					},
				},
			},
			out: "Revoking account...\n\n",
			err: testErr,
		},
		"revoke user failed": {
			cmd: RepoRevokeCommand{
				accountName: api.AccountName("dev1"),
				path:        "namespace/repo",
				force:       true,
			},
			userService: fakeclient.UserService{
				Getter: fakeclient.UserGetter{
					ReturnsUser: &api.User{
						AccountID: testUUID,
						Username:  "dev1",
						FullName:  "Developer Uno",
					},
				},
			},
			dirService: fakeclient.DirService{
				TreeGetter: fakeclient.TreeGetter{
					ReturnsTree: &api.Tree{
						RootDir: &api.Dir{
							Name: "repo",
						},
					},
				},
			},
			repoService: fakeclient.RepoService{
				UserService: &fakeclient.RepoUserService{
					Revoker: fakeclient.RepoRevoker{
						ReturnsRevokeResponse: &api.RevokeRepoResponse{
							Status: api.StatusFailed,
						},
					},
				},
			},
			// TODO SHDEV-1079: Fix this bug. When the revoke fails, the command should not print to stdout that
			// the dev "can no longer access the ... repository".
			out: "Revoking account...\n\n\n" +
				"Revoke failed! The account dev1 (Developer Uno) is the only admin on the repo namespace/repo." +
				"You need to make sure another account has admin rights on the repository or you can remove the repo." +
				"Revoke complete! The account dev1 (Developer Uno) can no longer access the namespace/repo repository. " +
				"Make sure you overwrite or delete all flagged secrets. " +
				"Secrets: 0 unaffected, 0 flagged\n",
		},
		// TODO SHDEV-1029: Add cases for confirm and abort after extracting AskForConfirmation out of ui.IO.
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			if tc.newClientErr != nil {
				tc.cmd.newClient = func() (secrethub.Client, error) {
					return nil, tc.newClientErr
				}
			} else {
				tc.cmd.newClient = func() (secrethub.Client, error) {
					return fakeclient.Client{
						AccountService: &tc.accountService,
						DirService:     &tc.dirService,
						RepoService:    &tc.repoService,
						ServiceService: &tc.serviceService,
						UserService:    &tc.userService,
					}, nil
				}
			}

			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Run
			err := tc.cmd.Run()

			// Assert
			assert.Equal(t, err, tc.err)
			assert.Equal(t, io.StdOut.String(), tc.out)
		})
	}
}
