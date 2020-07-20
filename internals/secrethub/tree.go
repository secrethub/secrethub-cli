package secrethub

import (
	"fmt"
	"io"
	"sort"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
)

// TreeCommand lists the contents of a directory at a given path in a tree-like format.
type TreeCommand struct {
	path          api.DirPath
	io            ui.IO
	fullPaths     bool
	noIndentation bool
	noReport      bool
	newClient     newClientFunc
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
		return err
	}

	t, err := client.Dirs().GetTree(cmd.path.Value(), -1, false)
	if err != nil {
		return err
	}

	cmd.printTree(t, cmd.io.Output())
	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *TreeCommand) Register(r command.Registerer) {
	clause := r.Command("tree", "List contents of a directory in a tree-like format.")
	clause.Arg("dir-path", "The path to to show contents for").Required().PlaceHolder(optionalDirPathPlaceHolder).SetValue(&cmd.path)

	clause.Flag("full-paths", "Print the full paths of the directories and secrets.").Short('f').BoolVar(&cmd.fullPaths)
	clause.Flag("no-indentation", "Print the content without indentation.").Short('i').BoolVar(&cmd.noIndentation)
	clause.Flag("no-report", "Skip the report at the bottom.").BoolVar(&cmd.noReport)

	command.BindAction(clause, cmd.Run)
}

// printTree recursively prints the tree's contents in a tree-like structure.
func (cmd *TreeCommand) printTree(t *api.Tree, w io.Writer) {
	name := colorizeByStatus(t.RootDir.Status, t.RootDir.Name)
	fmt.Fprintf(w, "%s\n", name)

	var format [4]string
	if cmd.noIndentation {
		format[0] = "%s%s%s\n"
		format[1] = "%s%s%s\n"
		format[2] = ""
		format[3] = ""
	} else {
		format[0] = "%s└── %s%s\n"
		format[1] = "%s├── %s%s\n"
		format[2] = "    "
		format[3] = "│   "
	}
	if cmd.fullPaths {
		cmd.printDirContentsRecursively(t.RootDir, "", w, format, t.RootDir.Name)
	} else {
		cmd.printDirContentsRecursively(t.RootDir, "", w, format, "")
	}
	if !cmd.noReport {
		fmt.Fprintf(w,
			"\n%s, %s\n",
			pluralize("directory", "directories", t.DirCount()),
			pluralize("secret", "secrets", t.SecretCount()),
		)
	}
}

// printDirContentsRecursively is a recursive function that prints the directory's contents
// in a tree-like structure, subdirs first followed by secrets.
func (cmd *TreeCommand) printDirContentsRecursively(dir *api.Dir, prefix string, w io.Writer, format [4]string, prevPath string) {

	sort.Sort(api.SortDirByName(dir.SubDirs))
	sort.Sort(api.SortSecretByName(dir.Secrets))

	total := len(dir.SubDirs) + len(dir.Secrets)

	if !cmd.fullPaths {
		prevPath = ""
	} else {
		prevPath += "/"
	}

	i := 0
	for _, sub := range dir.SubDirs {
		name := colorizeByStatus(sub.Status, sub.Name)

		if i == total-1 {
			fmt.Fprintf(w, format[0], prefix, prevPath, name)
			cmd.printDirContentsRecursively(sub, prefix+format[2], w, format, prevPath+name.(string))
		} else {
			fmt.Fprintf(w, format[1], prefix, prevPath, name)
			cmd.printDirContentsRecursively(sub, prefix+format[3], w, format, prevPath+name.(string))
		}
		i++
	}

	for _, secret := range dir.Secrets {
		name := colorizeByStatus(secret.Status, secret.Name)

		if i == total-1 {
			fmt.Fprintf(w, format[0], prefix, prevPath, name)
		} else {
			fmt.Fprintf(w, format[1], prefix, prevPath, name)
		}
		i++
	}
}
