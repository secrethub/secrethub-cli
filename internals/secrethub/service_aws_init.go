package secrethub

import (
	"fmt"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// Errors
var (
	ErrInvalidAWSRegion      = errMain.Code("invalid_region").Error("invalid AWS region")
	ErrRoleAlreadyTaken      = errMain.Code("role_taken").Error("a service using that IAM role already exists")
	ErrInvalidPermissionPath = errMain.Code("invalid_permission_path").ErrorPref("invalid permission path: %s")
)

// ServiceAWSInitCommand initializes a service for AWS.
type ServiceAWSInitCommand struct {
	description string
	repo        api.RepoPath
	kmsKeyID    string
	role        string
	region      string
	permission  string
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
		cmd.description = "AWS role " + roleNameFromRole(cmd.role)
	}

	service, err := client.Services().Create(cmd.repo.Value(), cmd.description, credentials.CreateAWS(cmd.kmsKeyID, cmd.role, cfg))
	if err == api.ErrCredentialAlreadyExists {
		return ErrRoleAlreadyTaken
	} else if err != nil {
		return err
	}

	err = givePermission(service, cmd.repo, cmd.permission, client)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "The service %s is now reachable through AWS when the role %s is assumed.\n", service.ServiceID, cmd.role)

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceAWSInitCommand) Register(r Registerer) {
	clause := r.Command("init", "Create a new AWS service account attached to a repository.")
	clause.Arg("repo", "The service account is attached to the repository in this path.").Required().SetValue(&cmd.repo)
	clause.Flag("kms-key-id", "ID of the KMS-key to be used for encrypting the service's account key.").Required().StringVar(&cmd.kmsKeyID)
	clause.Flag("role", "ARN of the IAM role that should have access to this service account.").Required().StringVar(&cmd.role)
	clause.Flag("region", "The AWS region that should be used").StringVar(&cmd.region)
	clause.Flag("description", "A description for the service so others will recognize it. Defaults to the name of the role that is attached to the service.").StringVar(&cmd.description)
	clause.Flag("descr", "").Hidden().StringVar(&cmd.description)
	clause.Flag("desc", "").Hidden().StringVar(&cmd.description)
	clause.Flag("permission", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use <permission> format to give permission on the root of the repo and <subdirectory>:<permission> to give permission on a subdirectory.").StringVar(&cmd.permission)

	BindAction(clause, cmd.Run)
}

// roleNameFromRole returns the name of the role indicated by the input. Accepted input is:
// - A role name (e.g. my-role)
// - A role name, prefixed by "role/" (e.g. role/my-role)
// - A role ARN (e.g. arn:aws:iam::123456789012:role/my-role)
//
// When the input is not one of these accepted inputs, no guarantees about the expected return
// are made.
func roleNameFromRole(role string) string {
	if strings.Contains(role, ":") {
		parts := strings.SplitN(role, "role/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	return strings.TrimPrefix(role, "role/")
}
