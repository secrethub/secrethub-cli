package secrethub

import (
	"bytes"
	"github.com/fatih/color"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

func TestSimpleTree(t *testing.T) {
	uuid0, _ := uuid.FromString("0")
	uuid1, _ := uuid.FromString("1")
	tree := &api.Tree{
		RootDir: &api.Dir{
			Name:  "test/repo",
			DirID: uuid0,
			SubDirs: []*api.Dir{
				{
					Name:     "secretFolder",
					DirID:    uuid1,
					ParentID: &uuid0,
					Secrets: []*api.Secret{
						{Name: "found you"},
					},
				},
			},
			Secrets: []*api.Secret{
				{Name: "mySecret"},
			},
		},
		Dirs: map[uuid.UUID]*api.Dir{
			uuid.New(): {
				DirID:    uuid0,
				ParentID: &uuid0,
			},
			uuid.New(): {
				Name:     "secretFolder",
				DirID:    uuid1,
				ParentID: &uuid0,
				Secrets: []*api.Secret{
					{Name: "found you"},
				},
			},
		},
		Secrets: map[uuid.UUID]*api.Secret{
			uuid.New(): {Name: "found you"},
			uuid.New(): {Name: "mySecret"},
		},
	}
	cases := map[string]struct {
		cmd            *TreeCommand
		expectedOutput string
	}{
		"simple tree": {
			cmd: &TreeCommand{
				io: ui.NewUserIO(),
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: "test/repo/\n" +
				"├── secretFolder/\n" +
				"│   └── found you\n" +
				"└── mySecret\n\n" +
				"1 directory, 2 secrets\n",
		},
		"full path": {
			cmd: &TreeCommand{
				path:      "test/repo",
				io:        ui.NewUserIO(),
				fullPaths: true,
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: "test/repo/\n" +
				"├── test/repo/secretFolder/\n" +
				"│   └── test/repo/secretFolder/found you\n" +
				"└── test/repo/mySecret\n\n" +
				"1 directory, 2 secrets\n",
		},
		"no indent": {
			cmd: &TreeCommand{
				io:            ui.NewUserIO(),
				noIndentation: true,
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: "test/repo/\n" +
				"secretFolder/\n" +
				"found you\n" +
				"mySecret\n\n" +
				"1 directory, 2 secrets\n",
		},
		"no report": {
			cmd: &TreeCommand{
				io:       ui.NewUserIO(),
				noReport: true,
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: "test/repo/\n" +
				"├── secretFolder/\n" +
				"│   └── found you\n" +
				"└── mySecret\n",
		},
		"all flags": {
			cmd: &TreeCommand{
				path:          "test/repo",
				io:            ui.NewUserIO(),
				fullPaths:     true,
				noIndentation: true,
				noReport:      true,
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: "test/repo/\n" +
				"test/repo/secretFolder/\n" +
				"test/repo/secretFolder/found you\n" +
				"test/repo/mySecret\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := &bytes.Buffer{}
			tc.cmd.printTree(tree, w)
			assert.Equal(t, w.String(), tc.expectedOutput)
		})
	}
}

func TestTreeColoring(t *testing.T) {
	color.NoColor = false
	uuid0, _ := uuid.FromString("0")
	uuid1, _ := uuid.FromString("1")
	uuid2, _ := uuid.FromString("2")
	tree := &api.Tree{
		RootDir: &api.Dir{
			Name:   "test/repo",
			Status: api.StatusFlagged,
			DirID:  uuid0,
			SubDirs: []*api.Dir{
				{
					Name:     "happy",
					DirID:    uuid1,
					ParentID: &uuid0,
				},
				{
					Name:     "secretFolder",
					DirID:    uuid2,
					ParentID: &uuid0,
					Status:   api.StatusFlagged,
					Secrets: []*api.Secret{
						{
							Name:   "found you",
							Status: api.StatusFlagged,
						},
					},
				},
			},
			Secrets: []*api.Secret{
				{Name: "mySecret"},
			},
		},
		Dirs: map[uuid.UUID]*api.Dir{
			uuid.New(): {
				DirID:    uuid0,
				ParentID: &uuid0,
			},
			uuid.New(): {
				Name:     "happy",
				DirID:    uuid1,
				ParentID: &uuid0,
			},
			uuid.New(): {
				Name:     "secretFolder",
				DirID:    uuid2,
				ParentID: &uuid0,
				Secrets: []*api.Secret{
					{Name: "found you"},
				},
			},
		},
		Secrets: map[uuid.UUID]*api.Secret{
			uuid.New(): {Name: "found you"},
			uuid.New(): {Name: "mySecret"},
		},
	}
	cases := map[string]struct {
		cmd            *TreeCommand
		expectedOutput string
	}{
		"simple tree": {
			cmd: &TreeCommand{
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: red.Sprint("test/repo/") + "\n" +
				"├── happy/\n" +
				"├── " + red.Sprint("secretFolder/") + "\n" +
				"│   └── " + red.Sprint("found you") + "\n" +
				"└── mySecret\n\n" +
				"2 directories, 2 secrets\n",
		},
		"full path": {
			cmd: &TreeCommand{
				path:      "test/repo",
				fullPaths: true,
				newClient: func() (secrethub.ClientInterface, error) {
					return &secrethub.Client{}, nil
				},
			},
			expectedOutput: red.Sprint("test/repo/") + "\n" +
				"├── test/repo/happy/\n" +
				"├── " + red.Sprint("test/repo/secretFolder/") + "\n" +
				"│   └── " + red.Sprint("test/repo/secretFolder/found you") + "\n" +
				"└── test/repo/mySecret\n\n" +
				"2 directories, 2 secrets\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := new(bytes.Buffer)
			tc.cmd.printTree(tree, w)
			assert.Equal(t, w.String(), tc.expectedOutput)
		})
	}
}
