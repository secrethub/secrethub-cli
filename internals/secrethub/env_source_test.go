package secrethub

import (
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
		newClient      newClientFunc
		expectedValues []string
		err            error
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
			err: errNameCollision{
				name:       "FOO_BAR",
				firstPath:  "namespace/repo/foo/bar",
				secondPath: "namespace/repo/foo_bar",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			source := newSecretsDirEnv(tc.newClient, dirPath)
			secrets, err := source.env()
			if tc.err != nil {
				assert.Equal(t, err, tc.err)
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
