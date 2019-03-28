package secrethub

import (
	"fmt"
	"sort"

	"io"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// TreeCommand lists the contents of a directory at a given path in a tree-like format.
type TreeCommand struct {
	path      api.DirPath
	io        ui.IO
	newClient newClientFunc
}

// NewTreeCommand creates a new TreeCommand.
func NewTreeCommand(io ui.IO, clientFactory newClientFunc) *TreeCommand {
	return &TreeCommand{
		io:        io,
		newClient: clientFactory,
	}
}

// Run prints the contents of a directory at a given path in a tree-like format.
func (cmd *TreeCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	t, err := client.Dirs().GetTree(cmd.path.Value(), -1, false)
	if err != nil {
		return errio.Error(err)
	}

	printTree(t, cmd.io.Stdout())
	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *TreeCommand) Register(r Registerer) {
	clause := r.Command("tree", "List contents of directories in a tree-like format.")
	clause.Arg("dir-path", "The path to to show contents for (<namespace>/<repo>[/<dir>])").Required().SetValue(&cmd.path)

	BindAction(clause, cmd.Run)
}

// printTree recursively prints the tree's contents in a tree-like structure.
func printTree(t *api.Tree, w io.Writer) {
	name := colorizeByStatus(t.RootDir.Status, t.RootDir.Name)
	fmt.Fprintf(w, "%s/\n", name)

	printDirContentsRecursively(t.RootDir, "", w)

	fmt.Fprintf(w,
		"\n%s, %s\n",
		pluralize("directory", "directories", t.DirCount()),
		pluralize("secret", "secrets", t.SecretCount()),
	)
}

// printDirContentsRecursively is a recursive function that prints the directory's contents
// in a tree-like structure, subdirs first followed by secrets.
func printDirContentsRecursively(dir *api.Dir, prefix string, w io.Writer) {

	sort.Sort(api.SortDirByName(dir.SubDirs))
	sort.Sort(api.SortSecretByName(dir.Secrets))

	total := len(dir.SubDirs) + len(dir.Secrets)

	i := 0
	for _, sub := range dir.SubDirs {
		name := colorizeByStatus(sub.Status, sub.Name)

		if i == total-1 {
			fmt.Fprintf(w, "%s└── %s/\n", prefix, name)
			printDirContentsRecursively(sub, prefix+"    ", w)
		} else {
			fmt.Fprintf(w, "%s├── %s/\n", prefix, name)
			printDirContentsRecursively(sub, prefix+"│   ", w)
		}
		i++
	}

	for _, secret := range dir.Secrets {
		name := colorizeByStatus(secret.Status, secret.Name)

		if i == total-1 {
			fmt.Fprintf(w, "%s└── %s\n", prefix, name)
		} else {
			fmt.Fprintf(w, "%s├── %s\n", prefix, name)
		}
		i++
	}
}
