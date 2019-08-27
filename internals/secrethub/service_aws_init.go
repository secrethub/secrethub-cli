package secrethub

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
)

// Errors
var (
	ErrInvalidAWSRegion = errMain.Code("invalid_region").Error("invalid AWS region")
	ErrRoleAlreadyTaken = errMain.Code("role_taken").Error("a service using that IAM role already exists")
)

// ServiceAWSInitCommand initializes a service for AWS.
type ServiceAWSInitCommand struct {
	description string
	path        api.DirPath
	kmsKeyID    string
	role        string
	region      string
	permission  api.Permission
	io          ui.IO
	newClient   newClientFunc
}

// NewServiceAWSInitCommand creates a new ServiceAWSInitCommand.
func NewServiceAWSInitCommand(io ui.IO, newClient newClientFunc) *ServiceAWSInitCommand {
	return &ServiceAWSInitCommand{
		io:        io,
		newClient: newClient,
	}
}

// Run initializes an AWS service.
func (cmd *ServiceAWSInitCommand) Run() error {
	repo := cmd.path.GetRepoPath()

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	cfg := aws.NewConfig()
	if cmd.region != "" {
		_, ok := endpoints.AwsPartition().Regions()[cmd.region]
		if !ok {
			return ErrInvalidAWSRegion
		}
		cfg = cfg.WithRegion(cmd.region)
	}

	if cmd.description == "" {
		cmd.description = cmd.role
	}

	service, err := client.Services().Create(repo.Value(), cmd.description, credentials.CreateAWS(cmd.kmsKeyID, cmd.role, cfg))
	if err == api.ErrCredentialAlreadyExists {
		return ErrRoleAlreadyTaken
	} else if err != nil {
		return err
	}

	if cmd.permission != 0 {
		_, err = client.AccessRules().Set(cmd.path.Value(), cmd.permission.String(), service.ServiceID)
		if err != nil {
			_, delErr := client.Services().Delete(service.ServiceID)
			if delErr != nil {
				fmt.Fprintf(cmd.io.Stdout(), "Failed to cleanup after creating an access rule for %s failed. Be sure to manually remove the created service account %s: %s\n", service.ServiceID, service.ServiceID, err)
				return errio.Error(delErr)
			}

			return errio.Error(err)
		}
	}

	fmt.Fprintf(cmd.io.Stdout(), "The service %s is now reachable through AWS when the role %s is assumed.\n", service.ServiceID, cmd.role)

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceAWSInitCommand) Register(r Registerer) {
	clause := r.Command("init", "Create a new AWS service account attached to a repository.")
	clause.Flag("path", "The service account is attached to the repository in this path and when used together with --permission, an access rule is created on the directory in this path.").Required().SetValue(&cmd.path)
	clause.Flag("kms-key-id", "ID of the KMS-key to be used for encrypting the service's account key.").Required().StringVar(&cmd.kmsKeyID)
	clause.Flag("role", "ARN of the IAM role that should have access to this service account.").Required().StringVar(&cmd.role)
	clause.Flag("region", "The AWS region that should be used").StringVar(&cmd.region)
	clause.Flag("desc", "A description for the service. Defaults to the ARN of the IAM role that is given access to the service.").StringVar(&cmd.description)
	clause.Flag("permission", "Automatically create an access rule giving the service account permission on the given path argument. Accepts `read`, `write` or `admin`.").SetValue(&cmd.permission)

	BindAction(clause, cmd.Run)
}
