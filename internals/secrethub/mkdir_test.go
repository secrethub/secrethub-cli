package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestMkDirCommand(t *testing.T) {
	cases := map[string]struct {
		paths     []string
		newClient func() (secrethub.ClientInterface, error)
		stdout    string
		err       error
	}{
		"success": {
			paths: []string{"namespace/repo/dir"},
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						CreateFunc: func(path string) (*api.Dir, error) {
							return &api.Dir{
								DirID:          uuid.New(),
								BlindName:      "blindname",
								Name:           "dir",
								Status:         api.StatusOK,
								CreatedAt:      time.Now().UTC(),
								LastModifiedAt: time.Now().UTC(),
							}, nil
						},
					},
				}, nil
			},
			stdout: "Created a new directory at namespace/repo/dir\n",
			err:    nil,
		},
		"success multiple dirs": {
			paths: []string{"namespace/repo/dir1", "namespace/repo/dir2"},
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						CreateFunc: func(path string) (*api.Dir, error) {
							return &api.Dir{
								DirID:          uuid.New(),
								BlindName:      "blindname",
								Name:           "dir",
								Status:         api.StatusOK,
								CreatedAt:      time.Now().UTC(),
								LastModifiedAt: time.Now().UTC(),
							}, nil
						},
					},
				}, nil
			},
			stdout: "Created a new directory at namespace/repo/dir1\nCreated a new directory at namespace/repo/dir2\n",
			err:    nil,
		},
		"new client fails": {
			paths: []string{"namespace/repo/dir"},
			newClient: func() (secrethub.ClientInterface, error) {
				return nil, errio.Namespace("test").Code("foo").Error("bar")
			},
			stdout: "",
			err:    errio.Namespace("test").Code("foo").Error("bar"),
		},
		"create dir fails": {
			paths: []string{"namespace/repo/dir"},
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						CreateFunc: func(path string) (*api.Dir, error) {
							return nil, api.ErrDirAlreadyExists
						},
					},
				}, nil
			},
			stdout: "",
			err:    api.ErrDirAlreadyExists,
		},
		"create dir fails on second dir": {
			paths: []string{"namespace/repo/dir1", "namespace/repo/dir2"},
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						CreateFunc: func(path string) (*api.Dir, error) {
							if path == "namespace/repo/dir2" {
								return nil, api.ErrDirAlreadyExists
							}
							return &api.Dir{
								DirID:          uuid.New(),
								BlindName:      "blindname",
								Name:           "dir",
								Status:         api.StatusOK,
								CreatedAt:      time.Now().UTC(),
								LastModifiedAt: time.Now().UTC(),
							}, nil
						},
					},
				}, nil
			},
			stdout: "Created a new directory at namespace/repo/dir1\n",
			err:    api.ErrDirAlreadyExists,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := ui.NewFakeIO()
			dirPaths := dirPathList{}
			for _, path := range tc.paths {
				_ = dirPaths.Set(path)
			}
			cmd := MkDirCommand{
				io:        io,
				paths:     dirPaths,
				newClient: tc.newClient,
			}

			err := cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, tc.stdout, io.StdOut.String())
		})
	}
}

func TestDirPathList_Set(t *testing.T) {
	cases := map[string]struct {
		path     string
		expected dirPathList
		err      error
	}{
		"success": {
			path:     "namespace/repo/dir",
			expected: dirPathList{"namespace/repo/dir"},
		},
		"root dir": {
			path: "namespace/repo",
			err:  ErrMkDirOnRootDir,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			list := dirPathList{}
			err := list.Set(tc.path)
			assert.Equal(t, err, tc.err)
			assert.Equal(t, len(list), len(tc.expected))
			for i := range list {
				assert.Equal(t, list[i], tc.expected[i])
			}
		})
	}
}
