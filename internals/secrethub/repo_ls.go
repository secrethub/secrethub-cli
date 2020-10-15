package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	// "github.com/spf13/cobra"
)

// RepoLSCommand lists repositories.
type RepoLSCommand struct {
	useTimestamps bool
	quiet         bool
	workspace     api.Namespace
	io            ui.IO
	timeFormatter TimeFormatter
	newClient     newClientFunc
}

// NewRepoLSCommand creates a new RepoLSCommand.
func NewRepoLSCommand(io ui.IO, newClient newClientFunc) *RepoLSCommand {
	return &RepoLSCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RepoLSCommand) Register(r cli.Registerer) {
	clause := r.Command("ls", "List all repositories you have access to.")
	clause.Alias("list")
	//// clause.Cmd.Args = cobra.MaximumNArgs(1)
	clause.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false, "Only print paths.")
	//clause.Arg("workspace", "When supplied, results are limited to repositories in this workspace.").SetValue(&cmd.workspace)
	registerTimestampFlag(clause, &cmd.useTimestamps)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.workspace})
}

// Run lists the repositories a user has access to.
func (cmd *RepoLSCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *RepoLSCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// run lists the repositories a user has access to.
func (cmd *RepoLSCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	var list []*api.Repo
	if cmd.workspace == "" {
		list, err = client.Repos().ListMine()
		if err != nil {
			return err
		}
	} else {
		list, err = client.Repos().List(cmd.workspace.String())
		if err != nil {
			return err
		}
	}

	sort.Sort(api.SortRepoByName(list))

	if cmd.quiet {
		for _, repo := range list {
			fmt.Fprintf(cmd.io.Output(), "%s\n", repo.Path())
		}
	} else {
		w := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\n", "NAME", "STATUS", "CREATED")
		for _, repo := range list {
			fmt.Fprintf(w, "%s\t%s\t%s\n", repo.Path(), repo.Status, cmd.timeFormatter.Format(repo.CreatedAt.Local()))
		}
		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}
