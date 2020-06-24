package secrethub

import (
	"fmt"
	"os/exec"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// ServiceGCPLinkCommand create a new link between a SecretHub namespace and a GCP project.
type ServiceGCPLinkCommand struct {
	namespace string
	projectID string
	io        ui.IO
	newClient newClientFunc
}

// NewServiceGCPLinkCommand creates a new ServiceGCPLinkCommand.
func NewServiceGCPLinkCommand(io ui.IO, newClient newClientFunc) *ServiceGCPLinkCommand {
	return &ServiceGCPLinkCommand{
		io:        io,
		newClient: newClient,
	}
}

func (cmd *ServiceGCPLinkCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	_, err = client.IDPLinks().GCP().Get(cmd.namespace, cmd.projectID)
	if !api.IsErrNotFound(err) {
		fmt.Fprintf(cmd.io.Output(), "Namespace %s and GCP project %s are already linked.\n", cmd.namespace, cmd.projectID)
		return nil
	}

	return createGCPLink(client, cmd.io, cmd.namespace, cmd.projectID)
}

func (cmd *ServiceGCPLinkCommand) Register(r command.Registerer) {
	clause := r.Command("link", "Create a new link between a namespace and a GCP project to allow creating SecretHub service accounts for GCP Service Accounts in the GCP project.")
	clause.Arg("namespace", "The SecretHub namespace to link.").Required().StringVar(&cmd.namespace)
	clause.Arg("project-id", "The GCP project to link the namespace to.").Required().StringVar(&cmd.projectID)

	command.BindAction(clause, cmd.Run)
}

// ServiceGCPListLinksCommand lists all existing links between the given namespace and GCP projects
type ServiceGCPListLinksCommand struct {
	namespace     string
	useTimestamps bool
	io            ui.IO
	newClient     newClientFunc
}

func NewServiceGCPListLinksCommand(io ui.IO, newClient newClientFunc) *ServiceGCPListLinksCommand {
	return &ServiceGCPListLinksCommand{
		io:        io,
		newClient: newClient,
	}
}

func (cmd *ServiceGCPListLinksCommand) Run() error {
	timeFormatter := NewTimeFormatter(cmd.useTimestamps)

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	links, err := client.IDPLinks().GCP().List(cmd.namespace)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\n", "PROJECT ID", "CREATED")

	for _, link := range links {
		_, err := fmt.Fprintf(tw, "%s\t%s\n", link.LinkedID, timeFormatter.Format(link.CreatedAt))
		if err != nil {
			return err
		}
	}

	err = tw.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (cmd *ServiceGCPListLinksCommand) Register(r command.Registerer) {
	clause := r.Command("list-links", "List all existing links between the given namespace and GCP projects.")
	clause.Arg("namespace", "The namespace for which to list all existing links to GCP projects.").Required().StringVar(&cmd.namespace)

	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// ServiceGCPDeleteLinkCommand deletes the link between a SecretHub namespace and a GCP project.
type ServiceGCPDeleteLinkCommand struct {
	namespace string
	projectID string
	io        ui.IO
	newClient newClientFunc
}

func NewServiceGCPDeleteLinkCommand(io ui.IO, newClient newClientFunc) *ServiceGCPDeleteLinkCommand {
	return &ServiceGCPDeleteLinkCommand{
		io:        io,
		newClient: newClient,
	}
}

func (cmd *ServiceGCPDeleteLinkCommand) Register(r command.Registerer) {
	clause := r.Command("delete-link", "Delete the link between a SecretHub namespace and a GCP project.")
	clause.Arg("namespace", "The SecretHub namespace to delete the link from.").Required().StringVar(&cmd.namespace)
	clause.Arg("project-id", "The GCP project to delete the link to.").Required().StringVar(&cmd.projectID)

	command.BindAction(clause, cmd.Run)
}

func (cmd *ServiceGCPDeleteLinkCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	_, err = client.IDPLinks().GCP().Get(cmd.namespace, cmd.projectID)
	if api.IsErrNotFound(err) {
		return err
	}

	question := fmt.Sprintf("Are you sure you want to delete the link link between GCP project %s and the namespace %s? Without the link, you cannot create new service accounts for this GCP project. This does not affect existing service accounts.", cmd.projectID, cmd.namespace)
	confirm, err := ui.AskYesNo(cmd.io, question, ui.DefaultNo)
	if err != nil {
		return err
	} else if !confirm {
		fmt.Println("Aborting.")
		return nil
	}

	return client.IDPLinks().GCP().Delete(cmd.namespace, cmd.projectID)
}

func createGCPLink(client secrethub.ClientInterface, io ui.IO, namespace, projectID string) error {
	l, err := client.IDPLinks().GCP().AuthorizationCodeListener()
	if err != nil {
		return fmt.Errorf("could not set up listener for authorization process: %s", err)
	}

	fmt.Fprintf(io.Output(), "If the browser does not automatically open, please go to the following link in your web browser: \n\n%s\n\n", l.AuthorizeURL())
	_ = openBrowser(l.AuthorizeURL())

	fmt.Fprint(io.Output(), "Waiting for you to complete authorization process...")
	progressPrinter := progress.NewPrinter(io.Output(), 2*time.Second)
	progressPrinter.Start()

	authorizationCode, err := l.WaitForAuthorizationCode()
	if err != nil {
		progressPrinter.Stop()
		return fmt.Errorf("could not retrieve authorization code: %s", err)
	}

	progressPrinter.Stop()

	_, err = client.IDPLinks().GCP().Create(namespace, projectID, authorizationCode, l.ListenURL())
	if err != nil {
		return err
	}

	fmt.Fprintf(io.Output(), "Created link between GCP project %s and SecretHub namespace %s\n", projectID, namespace)

	return nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
