package secrethub

import (
	"fmt"
	"os/exec"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/progress"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"
	"github.com/spf13/cobra"
)

// ServiceGCPLinkCommand create a new link between a SecretHub namespace and a GCP project.
type ServiceGCPLinkCommand struct {
	namespace api.OrgName
	projectID gcpProjectID
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

	exists, err := client.IDPLinks().GCP().Exists(cmd.namespace.String(), cmd.projectID.String())
	if err != nil {
		return err
	}
	if exists {
		fmt.Fprintf(cmd.io.Output(), "Namespace %s and GCP project %s are already linked.\n", cmd.namespace, cmd.projectID)
		return nil
	}

	return createGCPLink(client, cmd.io, cmd.namespace.String(), cmd.projectID.String())
}

func (cmd *ServiceGCPLinkCommand) Register(r command.Registerer) {
	clause := r.Command("link", "Create a new link between a namespace and a GCP project to allow creating SecretHub service accounts for GCP Service Accounts in the GCP project.")
	clause.Cmd.Args = cobra.ExactValidArgs(2)
	//clause.Arg("namespace", "The SecretHub namespace to link.").Required().SetValue(&cmd.namespace)
	//clause.Arg("project-id", "The GCP project to link the namespace to.").Required().SetValue(&cmd.projectID)

	clause.HelpLong("Linking a GCP project to a namespace is required to create SecretHub service accounts that use a GCP Service Account within the project. " +
		"A SecretHub namespace can be linked to multiple GCP projects and a GCP project can be linked to multiple namespaces.\n" +
		"\n" +
		"As long as the link exists, new service accounts can be created for the GCP project. " +
		"If a link is deleted, no new services can be created, but previously created services are unaffected. \n" +
		"\n" +
		"This command will open your browser where you are asked to authorize SecretHub to perform iam.test on your GCP projects. " +
		"This authorization is used to verify that you have access to the specified GCP project. " +
		"It is therefore important that the GCP account selected during the authorization process has access to the GCP project.\n" +
		"\n" +
		"Once the granted authorization has been used to confirm your access to the GCP project, the authorization will automatically be revoked. " +
		"This can be verified by going to https://myaccount.google.com/permissions. " +
		"Any reference to SecretHub should automatically disappear within a few minutes. " +
		"If it does not, the access can safely be revoked manually.")

	command.BindAction(clause, []cli.ArgValue{&cmd.namespace, &cmd.projectID}, cmd.Run)
}

// ServiceGCPListLinksCommand lists all existing links between the given namespace and GCP projects
type ServiceGCPListLinksCommand struct {
	namespace     api.Namespace
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

	tw := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\n", "PROJECT ID", "CREATED")

	iter := client.IDPLinks().GCP().List(cmd.namespace.String(), &secrethub.IdpLinkIteratorParams{})
	for {
		link, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		_, err = fmt.Fprintf(tw, "%s\t%s\n", link.LinkedID, timeFormatter.Format(link.CreatedAt))
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
	clause.Cmd.Args = cobra.ExactValidArgs(1)
	//clause.Arg("namespace", "The namespace for which to list all existing links to GCP projects.").Required().SetValue(&cmd.namespace)
	registerTimestampFlag(clause, &cmd.useTimestamps)

	command.BindAction(clause, []cli.ArgValue{&cmd.namespace}, cmd.Run)
}

// ServiceGCPDeleteLinkCommand deletes the link between a SecretHub namespace and a GCP project.
type ServiceGCPDeleteLinkCommand struct {
	namespace api.Namespace
	projectID gcpProjectID
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
	clause.HelpLong("After deleting the link you cannot create new GCP service accounts in the specified namespace and GCP project anymore. Exisiting service accounts will keep on working.")
	clause.Cmd.Args = cobra.ExactValidArgs(2)
	//clause.Arg("namespace", "The SecretHub namespace to delete the link from.").Required().SetValue(&cmd.namespace)
	//clause.Arg("project-id", "The GCP project to delete the link to.").Required().SetValue(&cmd.projectID)

	command.BindAction(clause, []cli.ArgValue{&cmd.namespace, &cmd.projectID}, cmd.Run)
}

func (cmd *ServiceGCPDeleteLinkCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	exists, err := client.IDPLinks().GCP().Exists(cmd.namespace.String(), cmd.projectID.String())
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("no existing link between GCP project %s and namespace %s found", cmd.projectID, cmd.namespace)
	}

	question := fmt.Sprintf("Are you sure you want to delete the link link between GCP project %s and the namespace %s? Without the link, you cannot create new service accounts for this GCP project. This does not affect existing service accounts.", cmd.projectID, cmd.namespace)
	confirm, err := ui.AskYesNo(cmd.io, question, ui.DefaultNo)
	if err != nil {
		return err
	} else if !confirm {
		fmt.Println("Aborting.")
		return nil
	}

	return client.IDPLinks().GCP().Delete(cmd.namespace.String(), cmd.projectID.String())
}

type gcpProjectID string

func (g *gcpProjectID) String() string {
	return string(*g)
}

func (g *gcpProjectID) Set(s string) error {
	if err := api.ValidateGCPProjectID(s); err != nil {
		return err
	}
	*g = gcpProjectID(s)
	return nil
}

func createGCPLink(client secrethub.ClientInterface, io ui.IO, namespace, projectID string) error {
	l, err := client.IDPLinks().GCP().AuthorizationCodeListener(namespace, projectID)
	if err != nil {
		return fmt.Errorf("could not set up listener for authorization process: %s", err)
	}

	fmt.Fprintf(io.Output(), "To create a link between the GCP project %s and the SecretHub namespace %s we have to verify your access to this GCP project. "+
		"After pressing [ENTER], a browser window will open and ask you to login to a Google account. "+
		"Please select an account that has read access to the GCP project. "+
		"You will then be asked to grant `Test IAM Permissions` permission to SecretHub. "+
		"This will be used to check whether you do have access to the project. "+
		"After this check has succeeded, the access will directly be revoked.\n\n", projectID, namespace)

	// If this fails, just continue.
	_, _ = ui.Ask(io, "Press [ENTER] to continue")

	fmt.Fprintf(io.Output(), "If the browser did not automatically open, please manually go to the following address in your browser: \n\n%s\n\n", l.AuthorizeURL())
	_ = openBrowser(l.AuthorizeURL())

	fmt.Fprint(io.Output(), "Waiting for you to complete authorization process...")
	progressPrinter := progress.NewPrinter(io.Output(), 2*time.Second)
	progressPrinter.Start()

	err = l.WithAuthorizationCode(func(authorizationCode string) error {
		_, err = client.IDPLinks().GCP().Create(namespace, projectID, authorizationCode, l.ListenURL())
		return err
	})
	if err != nil {
		progressPrinter.Stop()
		return err
	}

	progressPrinter.Stop()

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
