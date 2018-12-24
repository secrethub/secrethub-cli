package secrethub

import (
	"fmt"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	ErrMkDirOnRootDir = errMain.Code("mkdir_on_root_dir").Error("You cannot create a directory on the repo path. You can create subdirectories :owner/:repo_name/:directory_name.")
)

// MkDirCommand creates a new directory inside a repository.
type MkDirCommand struct {
	io        ui.IO
	path      api.DirPath
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
func (cmd *MkDirCommand) Register(r Registerer) {
	clause := r.Command("mkdir", "Create a new directory at a given path.")
	clause.Arg("dir-path", "The path to the directory (<namespace>/<repo>/<dir>[/<dir>])").Required().SetValue(&cmd.path)

	BindAction(clause, cmd.Run)
}

// Run executes the command.
func (cmd *MkDirCommand) Run() error {
	if cmd.path.IsRepoPath() {
		return ErrMkDirOnRootDir
	}

	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	_, err = client.Dirs().Create(cmd.path.Value())
	if err != nil {
		return errio.Error(err)
	}

	fmt.Fprintf(cmd.io.Stdout(), "Created a new directory at %s\n", cmd.path)

	return nil
}
