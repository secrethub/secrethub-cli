package secrethub

import (
	"errors"
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// ServiceGCPInitCommand initializes a service for GCP.
type ServiceGCPInitCommand struct {
	description         string
	repo                api.RepoPath
	kmsKeyResourceID    string
	serviceAccountEmail string
	permission          string
	io                  ui.IO
	newClient           newClientFunc
}

// NewServiceGCPInitCommand creates a new ServiceGCPInitCommand.
func NewServiceGCPInitCommand(io ui.IO, newClient newClientFunc) *ServiceGCPInitCommand {
	return &ServiceGCPInitCommand{
		io:        io,
		newClient: newClient,
	}
}

// Run initializes an GCP service.
func (cmd *ServiceGCPInitCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	if cmd.serviceAccountEmail == "" && cmd.kmsKeyResourceID == "" {
		fmt.Fprintln(cmd.io.Stdout(), "This command creates a new service account for use on GCP. For help on this, run `secrethub service gcp init --help`.")
	}

	if cmd.serviceAccountEmail == "" {
		serviceAccountEmail, err := ui.AskAndValidate(cmd.io, "What is the email of the GCP Service Account that should have access to the service?\n", 3, checkValidEmail)
		if err != nil {
			return err
		}
		cmd.serviceAccountEmail = strings.TrimSpace(serviceAccountEmail)
	}

	if cmd.kmsKeyResourceID == "" {
		kmsKey, err := ui.AskAndValidate(cmd.io, "What is the Resource ID of the KMS-key that should be used for encrypting the service's account key?\n", 3, checkIsNotEmpty("kms key"))
		if err != nil {
			return err
		}
		cmd.kmsKeyResourceID = strings.TrimSpace(kmsKey)
	}

	if cmd.description == "" {
		cmd.description = "GCP Service Account " + roleNameFromRole(cmd.serviceAccountEmail)
	}

	service, err := client.Services().Create(cmd.repo.Value(), cmd.description, credentials.CreateGCPServiceAccount(cmd.serviceAccountEmail, cmd.kmsKeyResourceID))
	if err == api.ErrCredentialAlreadyExists {
		return ErrRoleAlreadyTaken
	} else if err != nil {
		return err
	}

	if cmd.permission != "" {
		err = givePermission(service, cmd.repo, cmd.permission, client)
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.io.Stdout(), "Successfully created a new service account with ID: "+service.ServiceID)
	fmt.Fprintf(cmd.io.Stdout(), "Any host that assumes the Service Account %s can now automatically authenticate to SecretHub and fetch the secrets the service has been given access to.\n", cmd.serviceAccountEmail)

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceGCPInitCommand) Register(r command.Registerer) {
	clause := r.Command("init", "Create a new service account that is tied to an GCP IAM role.")
	clause.Arg("repo", "The service account is attached to the repository in this path.").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.repo)
	clause.Flag("kms-key", "The Resource ID of the KMS-key to be used for encrypting the service's account key.").StringVar(&cmd.kmsKeyResourceID)
	clause.Flag("service-account-email", "The email of the GCP Service Account that should have access to this service account.").StringVar(&cmd.serviceAccountEmail)
	clause.Flag("description", "A description for the service so others will recognize it. Defaults to the name of the role that is attached to the service.").StringVar(&cmd.description)
	clause.Flag("descr", "").Hidden().StringVar(&cmd.description)
	clause.Flag("desc", "").Hidden().StringVar(&cmd.description)
	clause.Flag("permission", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use `--permission <permission>` to give permission on the root of the repo and `--permission <dir>[/<dir> ...]:<permission>` to give permission on a subdirectory.").StringVar(&cmd.permission)

	clause.HelpLong("The native GCP identity provider uses a combination of GCP IAM and GCP KMS to provide access to SecretHub for any service running on GCP. For this to work, a GCP Service Account and a KMS key are needed.\n" +
		"\n" +
		"  - The GCP Service Account should be the service account that is assumed by the service during execution.\n" +
		"  - The KMS key is a key that is used for encryption of the account. Decryption permission on this key must be granted to the previously described GCP Service Account.\n" +
		"\n" +
		"To create a new service that uses the GCP identity provider, the CLI must have encryption access to the KMS key that will be used by the service account. Therefore GCP credentials should be configured on this system. For details on how this can be done, see https://cloud.google.com/sdk/docs/quickstarts.\n",
	)

	command.BindAction(clause, cmd.Run)
}

func checkValidEmail(v string) error {
	if !govalidator.IsEmail(v) {
		return errors.New("invalid email")
	}
	return nil
}
