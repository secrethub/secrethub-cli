package secrethub

import (
	"sort"
	"testing"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestSecretsDirEnv(t *testing.T) {
	const dirPath = "namespace/repo"
	rootDirUUID := uuid.New()
	subDirUUID := uuid.New()
	secretUUID1 := uuid.New()
	secretUUID2 := uuid.New()

	cases := map[string]struct {
		newClient          newClientFunc
		expectedValues     []string
		expectedCollission *errNameCollision
	}{
		"success": {
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: rootDirUUID,
									Name:  "repo",
								},
								Secrets: map[uuid.UUID]*api.Secret{
									secretUUID1: {
										SecretID: secretUUID1,
										DirID:    rootDirUUID,
										Name:     "foo",
									},
								},
							}, nil
						},
					},
				}, nil
			},
			expectedValues: []string{"FOO"},
		},
		"success secret in dir": {
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: rootDirUUID,
									Name:  "repo",
								},
								Dirs: map[uuid.UUID]*api.Dir{
									subDirUUID: {
										DirID:    subDirUUID,
										ParentID: &rootDirUUID,
										Name:     "foo",
									},
								},
								Secrets: map[uuid.UUID]*api.Secret{
									secretUUID1: {
										SecretID: secretUUID1,
										DirID:    subDirUUID,
										Name:     "bar",
									},
								},
							}, nil
						},
					},
				}, nil
			},
			expectedValues: []string{"FOO_BAR"},
		},
		"name collision": {
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: rootDirUUID,
									Name:  "repo",
								},
								Dirs: map[uuid.UUID]*api.Dir{
									subDirUUID: {
										DirID:    subDirUUID,
										ParentID: &rootDirUUID,
										Name:     "foo",
									},
								},
								Secrets: map[uuid.UUID]*api.Secret{
									secretUUID1: {
										SecretID: secretUUID1,
										DirID:    subDirUUID,
										Name:     "bar",
									},
									secretUUID2: {
										SecretID: secretUUID2,
										DirID:    rootDirUUID,
										Name:     "foo_bar",
									},
								},
							}, nil
						},
					},
				}, nil
			},
			expectedCollission: &errNameCollision{
				name: "FOO_BAR",
				paths: [2]string{
					"namespace/repo/foo/bar",
					"namespace/repo/foo_bar",
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			source := newSecretsDirEnv(tc.newClient, dirPath)
			secrets, err := source.env()
			if tc.expectedCollission != nil {
				collisionErr, ok := err.(errNameCollision)
				assert.Equal(t, ok, true)
				assert.Equal(t, collisionErr.name, tc.expectedCollission.name)

				gotPaths := collisionErr.paths[:]
				expectedPaths := tc.expectedCollission.paths[:]
				sort.Strings(gotPaths)
				sort.Strings(expectedPaths)

				assert.Equal(t, gotPaths, expectedPaths)
			} else {
				assert.OK(t, err)
				assert.Equal(t, len(secrets), len(tc.expectedValues))
				for _, name := range tc.expectedValues {
					if _, ok := secrets[name]; !ok {
						t.Errorf("expected but not found env var with name: %s", name)
					}
				}
			}
		})
	}
}
