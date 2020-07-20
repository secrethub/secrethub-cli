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
	path      api.DirPath
	io        ui.IO
	fullPaths bool
	noIndent  bool
	noReport  bool
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
	clause.Flag("full-paths", "Prints the full paths of the directories/secrets.").Short('f').BoolVar(&cmd.fullPaths)
	clause.Flag("no-indent", "Prints the tree structures without indentation.").Short('i').BoolVar(&cmd.noIndent)
	clause.Flag("no-report", "Does not print the directory and secret counts at the bottom").BoolVar(&cmd.noReport)

	command.BindAction(clause, cmd.Run)
}

// printTree recursively prints the tree's contents in a tree-like structure.
func (cmd *TreeCommand) printTree(t *api.Tree, w io.Writer) {
	name := colorizeByStatus(t.RootDir.Status, t.RootDir.Name)
	fmt.Fprintf(w, "%s/\n", name)

	var prefixes [4]string

	prefixes[0] = stringOrEmpty("└── ", !cmd.noIndent)
	prefixes[1] = stringOrEmpty("├── ", !cmd.noIndent)
	prefixes[2] = stringOrEmpty("│   ", !cmd.noIndent)
	prefixes[3] = stringOrEmpty("    ", !cmd.noIndent)

	cmd.printDirContentsRecursively(t.RootDir.Name, t.RootDir, "", w, prefixes)

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

func stringOrEmpty(string1 string, condition bool) string {
	if condition {
		return string1
	}
	return ""
}

func (cmd *TreeCommand) printDirContentsRecursively(priorPath string, dir *api.Dir, prefix string, w io.Writer, prefixes [4]string) {
	sort.Sort(api.SortDirByName(dir.SubDirs))
	sort.Sort(api.SortSecretByName(dir.Secrets))

	total := len(dir.SubDirs) + len(dir.Secrets)
	path := stringOrEmpty(priorPath+"/", cmd.fullPaths)

	i := 0
	for _, sub := range dir.SubDirs {
		name := colorizeByStatus(sub.Status, sub.Name)

		if i == total-1 {
			fmt.Fprintf(w, "%s%s%s/\n", prefix, prefixes[0], path+name.(string))
			cmd.printDirContentsRecursively(path+name.(string), sub, prefix+prefixes[3], w, prefixes)
		} else {
			fmt.Fprintf(w, "%s%s%s/\n", prefix, prefixes[1], path+name.(string))
			cmd.printDirContentsRecursively(path+name.(string), sub, prefix+prefixes[2], w, prefixes)
		}
		i++
	}

	for _, secret := range dir.Secrets {
		name := colorizeByStatus(secret.Status, secret.Name)

		if i == total-1 {
			fmt.Fprintf(w, "%s%s%s\n", prefix, prefixes[0], path+name.(string))
		} else {
			fmt.Fprintf(w, "%s%s%s\n", prefix, prefixes[1], path+name.(string))
		}
		i++
	}
}
