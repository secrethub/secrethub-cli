package secrethub

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"

	"github.com/secrethub/secrethub-go/internals/gcp"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/spf13/cobra"
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

	// Fail fast if the repo does not exist.
	_, err = client.Repos().Get(cmd.repo.String())
	if err != nil {
		return err
	}

	if cmd.serviceAccountEmail == "" && cmd.kmsKeyResourceID == "" {
		fmt.Fprintln(cmd.io.Stdout(), "This command creates a new service account for use on GCP. For help on this, run `secrethub service gcp init --help`.")

		var projectID string
		creds, err := transport.Creds(context.Background())
		if err == nil && creds.ProjectID != "" {
			projectID = creds.ProjectID
		} else {
			var projectLister gcpProjectOptionLister
			chosenProjectID, err := ui.ChooseDynamicOptions(cmd.io, "What GCP project do you want to use?", projectLister.Options, true, "project")
			if err != nil {
				return err
			}

			projectID = chosenProjectID
		}

		serviceAccountLister := gcpServiceAccountOptionLister{
			ProjectID: projectID,
		}
		serviceAccountEmail, err := ui.ChooseDynamicOptionsValidate(cmd.io, "What is the email of the service account you want to use?", serviceAccountLister.Options, "service account", api.ValidateGCPUserManagedServiceAccountEmail)
		if err != nil {
			return err
		}
		cmd.serviceAccountEmail = serviceAccountEmail

		kmsKeyLister, err := newGCPKeyOptionsLister(projectID)
		if err != nil {
			return err
		}
		keyring, err := ui.ChooseDynamicOptionsValidate(cmd.io, "In which keyring is the KMS key you want to use for encrypting the service account's key?", kmsKeyLister.KeyringOptions, "keyring", validateGCPKeyring)
		if err != nil {
			return err
		}
		kmsKey, err := ui.ChooseDynamicOptionsValidate(cmd.io, "What is the KMS key you want to use for encrypting the service account's key?", kmsKeyLister.KeyOptions(keyring), "kms key", validateGCPCryptoKey)
		if err != nil {
			return err
		}
		cmd.kmsKeyResourceID = kmsKey
	}

	if cmd.serviceAccountEmail == "" {
		serviceAccountEmail, err := ui.AskAndValidate(cmd.io, "What is the email of the GCP Service Account that should have access to the service?\n", 3, api.ValidateGCPUserManagedServiceAccountEmail)
		if err != nil {
			return err
		}
		cmd.serviceAccountEmail = strings.TrimSpace(serviceAccountEmail)
	}

	if cmd.kmsKeyResourceID == "" {
		kmsKey, err := ui.AskAndValidate(cmd.io, "What is the Resource ID of the KMS-key that should be used for encrypting the service's account key?\n", 3, api.ValidateGCPKMSKeyResourceID)
		if err != nil {
			return err
		}
		cmd.kmsKeyResourceID = strings.TrimSpace(kmsKey)
	}

	if cmd.description == "" {
		cmd.description = "GCP Service Account " + roleNameFromRole(cmd.serviceAccountEmail)
	}

	projectID, err := api.ProjectIDFromGCPEmail(cmd.serviceAccountEmail)
	if err != nil {
		return fmt.Errorf("invalid service account email: %s", err)
	}

	exists, err := client.IDPLinks().GCP().Exists(cmd.repo.GetNamespace(), projectID)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Fprintf(cmd.io.Output(), "This is the first time you're using a GCP Service Account in the GCP project %s for a SecretHub service account in the namespace %s. You have to link these two first.\n\n", projectID, cmd.repo.GetNamespace())

		err = createGCPLink(client, cmd.io, cmd.repo.GetNamespace(), projectID)
		if err != nil {
			return fmt.Errorf("could not create link: %s", err)
		}
	}

	service, err := client.Services().Create(cmd.repo.Value(), cmd.description, credentials.CreateGCPServiceAccount(cmd.serviceAccountEmail, cmd.kmsKeyResourceID))
	if err != nil {
		return err
	}

	if cmd.permission != "" {
		err = givePermission(service, cmd.repo, cmd.permission, client)
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.io.Stdout(), "Successfully created a new service account with ID: "+service.ServiceID)
	fmt.Fprintf(cmd.io.Stdout(), "Any host using the Service Account %s can now automatically authenticate to SecretHub and fetch the secrets the service has been given access to.\n", cmd.serviceAccountEmail)

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceGCPInitCommand) Register(r cli.Registerer) {
	clause := r.Command("init", "Create a new service account that is tied to a GCP Service Account.")
	clause.Cmd.Args = cobra.MaximumNArgs(1)
	//clause.Arg("repo", "The service account is attached to the repository in this path.").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.repo)
	clause.Flags().StringVar(&cmd.kmsKeyResourceID, "kms-key", "", "The Resource ID of the KMS-key to be used for encrypting the service's account key.")
	clause.Flags().StringVar(&cmd.serviceAccountEmail, "service-account-email", "", "The email of the GCP Service Account that should have access to this service account.")
	clause.Flags().StringVar(&cmd.description, "description", "", "A description for the service so others will recognize it. Defaults to the name of the role that is attached to the service.")
	clause.Flags().StringVar(&cmd.description, "descr", "", "")
	clause.Flags().StringVar(&cmd.description, "desc", "", "")
	clause.Cmd.Flag("desc").Hidden = true
	clause.Cmd.Flag("descr").Hidden = true
	clause.Flags().StringVar(&cmd.permission, "permission", "", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use `--permission <permission>` to give permission on the root of the repo and `--permission <dir>[/<dir> ...]:<permission>` to give permission on a subdirectory.")

	clause.HelpLong("The native GCP identity provider uses a combination of GCP IAM and GCP KMS to provide access to SecretHub for any service running on GCP. For this to work, a GCP Service Account and a KMS key are needed.\n" +
		"\n" +
		"  - The GCP Service Account should be the service account that is assumed by the service during execution.\n" +
		"  - The KMS key is a key that is used for encryption of the account. Decryption permission on this key must be granted to the previously described GCP Service Account.\n" +
		"\n" +
		"To create a new service that uses the GCP identity provider, the CLI must have encryption access to the KMS key that will be used by the service account. Therefore GCP application default credentials should be configured on this system. To achieve this, first install the Google Cloud SDK (https://cloud.google.com/sdk/docs/quickstarts) and then run `gcloud auth application-default login`.",
	)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.Argument{{Store: &cmd.repo, Name: "repo", Required: true}})
}

type gcpProjectOptionLister struct {
	nextPage string
}

func (l *gcpProjectOptionLister) Options() ([]ui.Option, bool, error) {
	// Explicitly setting the credentials is needed to avoid a permission denied error from cloudresourcemanager.
	creds, err := transport.Creds(context.Background())
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}

	crm, err := cloudresourcemanager.NewService(context.Background(), option.WithTokenSource(creds.TokenSource))
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}

	resp, err := crm.Projects.List().Filter("lifecycleState:ACTIVE").PageToken(l.nextPage).PageSize(10).Do()
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}

	options := make([]ui.Option, len(resp.Projects))
	for i, project := range resp.Projects {
		options[i] = ui.Option{
			Value:   project.ProjectId,
			Display: fmt.Sprintf("%s (%s)", project.Name, project.ProjectId),
		}
	}

	l.nextPage = resp.NextPageToken
	return options, resp.NextPageToken == "", nil
}

type gcpServiceAccountOptionLister struct {
	ProjectID string
	nextPage  string
}

func (l *gcpServiceAccountOptionLister) Options() ([]ui.Option, bool, error) {
	iamService, err := iam.NewService(context.Background())
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}

	resp, err := iamService.Projects.ServiceAccounts.List("projects/" + l.ProjectID).PageToken(l.nextPage).PageSize(10).Do()
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}

	options := make([]ui.Option, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		// Only list user-managed service accounts
		if err := api.ValidateGCPUserManagedServiceAccountEmail(account.Email); err != nil {
			continue
		}
		display := account.Email
		if account.Description != "" {
			display += " (" + account.Description + ")"
		}
		options = append(options, ui.Option{
			Value:   account.Email,
			Display: display,
		})
	}

	l.nextPage = resp.NextPageToken
	return options, resp.NextPageToken == "", nil
}

func newGCPKeyOptionsLister(projectID string) (*gcpKMSKeyOptionLister, error) {
	kmsService, err := cloudkms.NewService(context.Background())
	if err != nil {
		return nil, gcp.HandleError(err)
	}

	return &gcpKMSKeyOptionLister{
		projectID:  projectID,
		kmsService: kmsService,
	}, nil
}

type gcpKMSKeyOptionLister struct {
	projectID  string
	nextPage   string
	kmsService *cloudkms.Service
}

func (l *gcpKMSKeyOptionLister) KeyringOptions() ([]ui.Option, bool, error) {
	options := make([]ui.Option, 0, 16)

	errChan := make(chan error, 1)
	resChan := make(chan ui.Option, 16)
	var wg sync.WaitGroup

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelTimeout()

	err := l.kmsService.Projects.Locations.List("projects/"+l.projectID).Pages(ctx, func(resp *cloudkms.ListLocationsResponse) error {
		for _, loc := range resp.Locations {
			wg.Add(1)
			go func(locationName string) {
				err := l.kmsService.Projects.Locations.KeyRings.List(locationName).Pages(ctx, func(resp *cloudkms.ListKeyRingsResponse) error {
					for _, keyring := range resp.KeyRings {
						resChan <- ui.Option{
							Value:   keyring.Name,
							Display: keyring.Name,
						}
					}
					return nil
				})
				wg.Done()
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
			}(loc.Name)
		}
		return nil
	})
	if err != nil {
		return nil, false, gcp.HandleError(err)
	}
	go func() {
		wg.Wait()
		close(resChan)
	}()
	for res := range resChan {
		options = append(options, res)
	}
	select {
	case err := <-errChan:
		return nil, false, gcp.HandleError(err)
	default:
		return options, false, nil
	}
}

func validateGCPKeyring(keyring string) error {
	if !regexp.MustCompile("^projects/[a-zA-Z0-9-]+/locations/[a-zA-Z0-9-]+/keyRings/[a-zA-Z0-9-_]+$").MatchString(keyring) {
		return errors.New("GCP keyring should be in the form \"projects/<project-id>/locations/<location>/keyRings/<key-ring>\"")
	}
	return nil
}

func validateGCPCryptoKey(cryptoKey string) error {
	if !regexp.MustCompile("^projects/[a-zA-Z0-9-]+/locations/[a-zA-Z0-9-]+/keyRings/[a-zA-Z0-9-_]+/cryptoKeys/[a-zA-Z0-9-_]+$").MatchString(cryptoKey) {
		return errors.New("GCP crypto key should be in the form \"projects/<project-id>/locations/<location>/keyRings/<key-ring>/cryptoKeys/<key>\"")
	}
	return nil
}

func (l *gcpKMSKeyOptionLister) KeyOptions(keyring string) func() ([]ui.Option, bool, error) {
	return func() ([]ui.Option, bool, error) {
		resp, err := l.kmsService.Projects.Locations.KeyRings.CryptoKeys.List(keyring).PageSize(10).Filter("purpose:ENCRYPT_DECRYPT").PageToken(l.nextPage).Do()
		if err != nil {
			return nil, false, gcp.HandleError(err)
		}

		options := make([]ui.Option, len(resp.CryptoKeys))
		for i, key := range resp.CryptoKeys {
			options[i] = ui.Option{
				Value:   key.Name,
				Display: key.Name,
			}
		}

		l.nextPage = resp.NextPageToken
		return options, resp.NextPageToken == "", nil
	}
}
