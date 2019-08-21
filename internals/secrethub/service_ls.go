package secrethub

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
)

// ServiceLsCommand lists all service accounts in a given repository.
type ServiceLsCommand struct {
	repoPath api.RepoPath
	quiet    bool

	io           ui.IO
	newClient    newClientFunc
	serviceTable serviceTable
}

// NewServiceLsCommand creates a new ServiceLsCommand.
func NewServiceLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:           io,
		newClient:    newClient,
		serviceTable: keyServiceTable{},
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

		fmt.Fprintln(w, strings.Join(cmd.serviceTable.header(), "\t"))

		for _, service := range services {
			fmt.Fprintln(w, strings.Join(cmd.serviceTable.row(service), "\t"))
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

type serviceTable interface {
	header() []string
	row(service *api.Service) []string
}

type baseServiceTable struct{}

func (sw baseServiceTable) header() []string {
	return []string{"ID", "DESCRIPTION"}
}

func (sw baseServiceTable) row(service *api.Service) []string {
	return []string{service.ServiceID, service.Description}
}

type keyServiceTable struct {
	baseServiceTable
}

func (sw keyServiceTable) header() []string {
	return append(sw.baseServiceTable.header(), "TYPE")
}

func (sw keyServiceTable) row(service *api.Service) []string {
	return append(sw.baseServiceTable.row(service), string(service.Credential.Type))
}
