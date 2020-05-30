package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

// Errors
var (
	ErrMkDirOnRootDir = errMain.Code("mkdir_on_root_dir").Error("You cannot create a directory on the repo path. You can create subdirectories :owner/:repo_name/:directory_name.")
)

// MkDirCommand creates a new directory inside a repository.
type MkDirCommand struct {
	io        ui.IO
	paths     dirPathList
	parents   bool
	newClient newClientFunc
}

// NewMkDirCommand returns a new command.
func NewMkDirCommand(io ui.IO, newClient newClientFunc) *MkDirCommand {
	return &MkDirCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *MkDirCommand) Register(r command.Registerer) {
	clause := r.Command("mkdir", "Create a new directory.")
	clause.Arg("dir-paths", "The paths to the directories").Required().PlaceHolder(dirPathsPlaceHolder).SetValue(&cmd.paths)
	clause.Flag("parents", "Create parent directories if needed. Does not error when directories already exist.").BoolVar(&cmd.parents)

	command.BindAction(clause, cmd.Run)
}

// Run executes the command.
func (cmd *MkDirCommand) Run() error {
	for _, path := range cmd.paths {
		if path.IsRepoPath() {
			return ErrMkDirOnRootDir
		}
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	for _, path := range cmd.paths {
		if cmd.parents {
			err = client.Dirs().CreateAll(path.Value())
			if err != nil {
				return err
			}
		} else {
			_, err = client.Dirs().Create(path.Value())
			if err != nil {
				return err
			}
		}

		fmt.Fprintf(cmd.io.Stdout(), "Created a new directory at %s\n", path)
	}
	return nil
}

type dirPathList []api.DirPath

func (d *dirPathList) String() string {
	return ""
}

func (d *dirPathList) Set(path string) error {
	dirPath, err := api.NewDirPath(path)
	if err != nil {
		return err
	}
	*d = append(*d, dirPath)
	return nil
}

func (d *dirPathList) IsCumulative() bool {
	return true
}
