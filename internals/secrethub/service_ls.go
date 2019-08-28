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

	io              ui.IO
	useTimestamps   bool
	newClient       newClientFunc
	newServiceTable func(t TimeFormatter) serviceTable
}

// NewServiceLsCommand creates a new ServiceLsCommand.
func NewServiceLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:              io,
		newClient:       newClient,
		newServiceTable: newKeyServiceTable,
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
		serviceTable := cmd.newServiceTable(NewTimeFormatter(cmd.useTimestamps))

		fmt.Fprintln(w, strings.Join(serviceTable.header(), "\t"))

		for _, service := range services {
			fmt.Fprintln(w, strings.Join(serviceTable.row(service), "\t"))
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

type baseServiceTable struct {
	timeFormatter TimeFormatter
}

func (sw baseServiceTable) header() []string {
	return []string{"ID", "DESCRIPTION", "CREATED"}
}

func (sw baseServiceTable) row(service *api.Service) []string {
	return []string{service.ServiceID, service.Description, sw.timeFormatter.Format(service.CreatedAt.Local())}
}

func newKeyServiceTable(timeFormatter TimeFormatter) serviceTable {
	return keyServiceTable{baseServiceTable{timeFormatter: timeFormatter}}
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
