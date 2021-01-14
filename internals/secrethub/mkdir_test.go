package secrethub

import (
	"testing"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

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
		},
		"create dir fails on first dir": {
			paths: []string{"namespace/repo/dir1", "namespace/repo/dir2"},
			newClient: func() (secrethub.ClientInterface, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						CreateFunc: func(path string) (*api.Dir, error) {
							if path == "namespace/repo/dir1" {
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
			stdout: "Created a new directory at namespace/repo/dir2\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := fakeui.NewIO(t)
			dirPaths := cli.StringListValue{}
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
			assert.Equal(t, tc.stdout, io.Out.String())
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	cases := map[string]struct {
		client secrethub.ClientInterface
		path   string
		err    error
	}{
		"success": {
			client: fakeclient.Client{
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
			},
			path: "namespace/repo/dir",
		},
		"root dir": {
			client: fakeclient.Client{
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
			},
			path: "namespace/repo",
			err:  ErrMkDirOnRootDir,
		},
		"create dir fails": {
			client: fakeclient.Client{
				DirService: &fakeclient.DirService{
					CreateFunc: func(path string) (*api.Dir, error) {
						return nil, api.ErrDirAlreadyExists
					},
				},
			},
			path: "namespace/repo/dir",
			err:  api.ErrDirAlreadyExists,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := MkDirCommand{}
			err := cmd.createDirectory(tc.client, tc.path)
			assert.Equal(t, err, tc.err)
		})
	}
}
