package secrethub

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/spf13/cobra"
)

// ServiceLsCommand lists all service accounts in a given repository.
type ServiceLsCommand struct {
	repoPath api.RepoPath
	quiet    bool

	io              ui.IO
	useTimestamps   bool
	newClient       newClientFunc
	newServiceTable func(t TimeFormatter) serviceTable
	filters         []func(service *api.Service) bool
	help            string
}

// NewServiceLsCommand creates a new ServiceLsCommand.
func NewServiceLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:              io,
		newClient:       newClient,
		newServiceTable: newKeyServiceTable,
		help:            "List all service accounts in a given repository.",
	}
}

func NewServiceAWSLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:              io,
		newClient:       newClient,
		newServiceTable: newAWSServiceTable,
		filters: []func(service *api.Service) bool{
			isAWSService,
		},
		help: "List all AWS service accounts in a given repository.",
	}
}

func NewServiceGCPLsCommand(io ui.IO, newClient newClientFunc) *ServiceLsCommand {
	return &ServiceLsCommand{
		io:              io,
		newClient:       newClient,
		newServiceTable: newGCPServiceTable,
		filters: []func(service *api.Service) bool{
			isGCPService,
		},
		help: "List all GCP service accounts in a given repository.",
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceLsCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("ls", cmd.help)
	clause.Alias("list")
	clause.Args = cobra.ExactValidArgs(1)
	//clause.Arg("repo-path", "The path to the repository to list services for").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.repoPath)
	clause.BoolVarP(&cmd.quiet, "quiet", "q", false, "Only print service IDs.")
	registerTimestampFlag(clause, &cmd.useTimestamps)

	command.BindAction(clause, cmd.argumentRegister, cmd.Run)
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

	included := []*api.Service{}
outer:
	for _, service := range services {
		for _, filter := range cmd.filters {
			if !filter(service) {
				continue outer
			}
		}
		included = append(included, service)
	}

	if cmd.quiet {
		for _, service := range included {
			fmt.Fprintf(cmd.io.Output(), "%s\n", service.ServiceID)
		}
	} else {
		w := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)
		serviceTable := cmd.newServiceTable(NewTimeFormatter(cmd.useTimestamps))

		fmt.Fprintln(w, strings.Join(serviceTable.header(), "\t"))

		for _, service := range included {
			fmt.Fprintln(w, strings.Join(serviceTable.row(service), "\t"))
		}

		err = w.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

func (cmd *ServiceLsCommand) argumentRegister(c *cobra.Command, args []string) error {
	var err error
	cmd.repoPath, err = api.NewRepoPath(args[0])
	if err != nil {
		return err
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

func (sw baseServiceTable) header(content ...string) []string {
	res := append([]string{"ID", "DESCRIPTION"}, content...)
	return append(res, "CREATED")
}

func (sw baseServiceTable) row(service *api.Service, content ...string) []string {
	res := append([]string{service.ServiceID, service.Description}, content...)
	return append(res, sw.timeFormatter.Format(service.CreatedAt.Local()))
}

func newKeyServiceTable(timeFormatter TimeFormatter) serviceTable {
	return keyServiceTable{baseServiceTable{timeFormatter: timeFormatter}}
}

type keyServiceTable struct {
	baseServiceTable
}

func (sw keyServiceTable) header() []string {
	return sw.baseServiceTable.header("TYPE")
}

func (sw keyServiceTable) row(service *api.Service) []string {
	return sw.baseServiceTable.row(service, string(service.Credential.Type))
}

func newAWSServiceTable(timeFormatter TimeFormatter) serviceTable {
	return awsServiceTable{baseServiceTable{timeFormatter: timeFormatter}}
}

type awsServiceTable struct {
	baseServiceTable
}

func (sw awsServiceTable) header() []string {
	return sw.baseServiceTable.header("ROLE", "KMS-KEY")
}

func (sw awsServiceTable) row(service *api.Service) []string {
	return sw.baseServiceTable.row(service, service.Credential.Metadata[api.CredentialMetadataAWSRole], service.Credential.Metadata[api.CredentialMetadataAWSKMSKey])
}

func isAWSService(service *api.Service) bool {
	if service == nil {
		return false
	}

	return service.Credential.Type == api.CredentialTypeAWS
}

type gcpServiceTable struct {
	baseServiceTable
}

func newGCPServiceTable(timeFormatter TimeFormatter) serviceTable {
	return gcpServiceTable{baseServiceTable{timeFormatter: timeFormatter}}
}

func (sw gcpServiceTable) header() []string {
	return sw.baseServiceTable.header("SERVICE-ACCOUNT-EMAIL", "KMS-KEY")
}

func (sw gcpServiceTable) row(service *api.Service) []string {
	return sw.baseServiceTable.row(service, service.Credential.Metadata[api.CredentialMetadataGCPServiceAccountEmail], service.Credential.Metadata[api.CredentialMetadataGCPKMSKeyResourceID])
}

func isGCPService(service *api.Service) bool {
	if service == nil {
		return false
	}

	return service.Credential.Type == api.CredentialTypeGCPServiceAccount
}
