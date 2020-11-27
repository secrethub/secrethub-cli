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

	clause.Flag("full-paths", "Print the full path of each directory and secret.").Short('f').BoolVar(&cmd.fullPaths)
	clause.Flag("no-indentation", "Don't print indentation lines.").Short('i').BoolVar(&cmd.noIndentation)
	clause.Flag("no-report", "Turn off secret/directory count at end of tree listing.").BoolVar(&cmd.noReport)
	clause.Flag("noreport", "Turn off secret/directory count at end of tree listing.").Hidden().BoolVar(&cmd.noReport)

	command.BindAction(clause, cmd.Run)
}

// printTree recursively prints the tree's contents in a tree-like structure.
func (cmd *TreeCommand) printTree(t *api.Tree, w io.Writer) {
	printTree(t, w, cmd.fullPaths, cmd.path.Value(), !cmd.noReport, !cmd.noIndentation)
}

func printTree(t *api.Tree, w io.Writer, fullPaths bool, path string, showReport bool, indentation bool) {
	rootDirName := func() string {
		if fullPaths {
			return path + "/"
		}
		return t.RootDir.Name + "/"
	}()
	name := colorizeByStatus(t.RootDir.Status, rootDirName)
	fmt.Fprintf(w, "%s\n", name)

	prevPath := ""
	if fullPaths {
		prevPath = path
	}
	printDirContentsRecursively(t.RootDir, "", w, prevPath, fullPaths, indentation)

	if showReport {
		fmt.Fprintf(w,
			"\n%s, %s\n",
			pluralize("directory", "directories", t.DirCount()),
			pluralize("secret", "secrets", t.SecretCount()),
		)
	}
}

// printDirContentsRecursively is a recursive function that prints the directory's contents
// in a tree-like structure, subdirs first followed by secrets.
func printDirContentsRecursively(dir *api.Dir, prefix string, w io.Writer, prevPath string, fullPaths bool, indentation bool) {

	sort.Sort(api.SortDirByName(dir.SubDirs))
	sort.Sort(api.SortSecretByName(dir.Secrets))

	total := len(dir.SubDirs) + len(dir.Secrets)

	if fullPaths {
		prevPath += "/"
	} else {
		prevPath = ""
	}

	i := 0
	for _, sub := range dir.SubDirs {

		name := sub.Name
		if fullPaths {
			name = prevPath + name
		}
		colorName := colorizeByStatus(sub.Status, name+"/")

		if !indentation {
			fmt.Fprintf(w, "%s\n", colorName)
			printDirContentsRecursively(sub, prefix, w, name, fullPaths, indentation)
		} else if i == total-1 {
			fmt.Fprintf(w, "%s└── %s\n", prefix, colorName)
			printDirContentsRecursively(sub, prefix+"    ", w, name, fullPaths, indentation)
		} else {
			fmt.Fprintf(w, "%s├── %s\n", prefix, colorName)
			printDirContentsRecursively(sub, prefix+"│   ", w, name, fullPaths, indentation)
		}
		i++
	}

	for _, secret := range dir.Secrets {
		name := secret.Name
		if fullPaths {
			name = prevPath + name
		}
		colorName := colorizeByStatus(secret.Status, name)

		if !indentation {
			fmt.Fprintf(w, "%s\n", colorName)
		} else if i == total-1 {
			fmt.Fprintf(w, "%s└── %s\n", prefix, colorName)
		} else {
			fmt.Fprintf(w, "%s├── %s\n", prefix, colorName)
		}
		i++
	}
}
