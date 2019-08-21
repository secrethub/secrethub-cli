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

	io        ui.IO
	newClient newClientFunc
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

	BindAction(clause, cmd.Run)
}

// Run lists all service accounts in a given repository.
func (cmd *ServiceLsCommand) Run() error {
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

		fmt.Fprintf(w, "%s\t%s\n", "ID", "DESCRIPTION")

		for _, service := range services {
			fmt.Fprintf(w, "%s\t%s\n", service.ServiceID, service.Description)
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}
