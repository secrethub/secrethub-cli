package secrethub

import (
	"fmt"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
)

// ServiceLsCommand lists all service accounts in a given repository.
type ServiceLsCommand struct {
	repoPath api.RepoPath
	quiet    bool

	io            ui.IO
	timeFormatter TimeFormatter
	useTimestamps bool
	newClient     newClientFunc
}

// NewServiceLsCommand creates a new ServiceLsCommand.
func NewServiceLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceLsCommand) Register(r Registerer) {
	clause := r.Command("ls", "List all service accounts in a given repository.")
	clause.Arg("repo-path", "The path to the repository to list services for (<namespace>/<repo>).").Required().SetValue(&cmd.repoPath)
	clause.Flag("quiet", "Only print service IDs.").Short('q').BoolVar(&cmd.quiet)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	BindAction(clause, cmd.Run)
}

// Run lists all service accounts in a given repository.
func (cmd *ServiceLsCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *ServiceLsCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// Run lists all service accounts in a given repository.
func (cmd *ServiceLsCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	services, err := client.Services().List(cmd.repoPath.Value())
	if err != nil {
		return err
	}

	if cmd.quiet {
		for _, service := range services {
			fmt.Fprintf(cmd.io.Stdout(), "%s\n", service.ServiceID)
		}
	} else {
		w := tabwriter.NewWriter(cmd.io.Stdout(), 0, 2, 2, ' ', 0)

		fmt.Fprintf(w, "%s\t%s\t%s\n", "ID", "DESCRIPTION", "CREATED")

		for _, service := range services {
			fmt.Fprintf(w, "%s\t%s\t%s\n", service.ServiceID, service.Description, cmd.timeFormatter.Format(service.CreatedAt.Local()))
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}
