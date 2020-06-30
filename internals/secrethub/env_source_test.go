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
	testUUID1 := uuid.New()
	testUUID2 := uuid.New()
	testUUID3 := uuid.New()
	testUUID4 := uuid.New()

	cases := map[string]struct {
		clientFunc     newClientFunc
		expectedValues []string
		err            error
	}{
		"success": {
			clientFunc: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: testUUID1,
									Name:  "repo",
								},
								Secrets: map[uuid.UUID]*api.Secret{
									testUUID2: {
										SecretID: testUUID2,
										DirID:    testUUID1,
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
			clientFunc: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: testUUID1,
									Name:  "repo",
								},
								Dirs: map[uuid.UUID]*api.Dir{
									testUUID2: {
										DirID:    testUUID2,
										ParentID: &testUUID1,
										Name:     "foo",
									},
								},
								Secrets: map[uuid.UUID]*api.Secret{
									testUUID3: {
										SecretID: testUUID3,
										DirID:    testUUID2,
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
			clientFunc: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
							return &api.Tree{
								ParentPath: "namespace",
								RootDir: &api.Dir{
									DirID: testUUID1,
									Name:  "repo",
								},
								Dirs: map[uuid.UUID]*api.Dir{
									testUUID2: {
										DirID:    testUUID2,
										ParentID: &testUUID1,
										Name:     "foo",
									},
								},
								Secrets: map[uuid.UUID]*api.Secret{
									testUUID3: {
										SecretID: testUUID3,
										DirID:    testUUID2,
										Name:     "bar",
									},
									testUUID4: {
										SecretID: testUUID4,
										DirID:    testUUID1,
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
			source := newSecretsDirEnv(tc.clientFunc, dirPath)
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
