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
		path      string
		newClient func() (secrethub.ClientAdapter, error)
		stdout    string
		err       error
	}{
		"success": {
			path: "namespace/repo/dir",
			newClient: func() (secrethub.ClientAdapter, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						Creater: fakeclient.DirCreater{
							ReturnsDir: &api.Dir{
								DirID:          uuid.New(),
								BlindName:      "blindname",
								Name:           "dir",
								Status:         api.StatusOK,
								CreatedAt:      time.Now().UTC(),
								LastModifiedAt: time.Now().UTC(),
							},
							Err: nil,
						},
					},
				}, nil
			},
			stdout: "Created a new directory at namespace/repo/dir\n",
			err:    nil,
		},
		"on root dir": {
			path:   "namespace/repo",
			stdout: "",
			err:    ErrMkDirOnRootDir,
		},
		"new client fails": {
			path: "namespace/repo/dir",
			newClient: func() (secrethub.ClientAdapter, error) {
				return nil, errio.Namespace("test").Code("foo").Error("bar")
			},
			stdout: "",
			err:    errio.Namespace("test").Code("foo").Error("bar"),
		},
		"create dir fails": {
			path: "namespace/repo/dir",
			newClient: func() (secrethub.ClientAdapter, error) {
				return fakeclient.Client{
					DirService: &fakeclient.DirService{
						Creater: fakeclient.DirCreater{
							Err: api.ErrDirAlreadyExists,
						},
					},
				}, nil
			},
			stdout: "",
			err:    api.ErrDirAlreadyExists,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := ui.NewFakeIO()
			cmd := MkDirCommand{
				io:        io,
				path:      api.DirPath(tc.path),
				newClient: tc.newClient,
			}

			err := cmd.Run()

			assert.Equal(t, err, tc.err)
			assert.Equal(t, tc.stdout, io.StdOut.String())
		})
	}
}
