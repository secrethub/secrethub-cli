package secrethub

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	shaws "github.com/secrethub/secrethub-go/internals/aws"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/cobra"
)

// Errors
var (
	ErrInvalidAWSRegion      = errMain.Code("invalid_region").Error("invalid AWS region")
	ErrInvalidPermissionPath = errMain.Code("invalid_permission_path").ErrorPref("invalid permission path: %s")
	ErrMissingRegion         = errMain.Code("missing_region").Error("could not find AWS region. Supply using the --region flag or in the AWS configuration. See https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html for the AWS configuration files")
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

	if cmd.role == "" && cmd.kmsKeyID == "" {
		fmt.Fprintln(cmd.io.Output(), "This command creates a new service account for use on AWS. For help on this, run `secrethub service aws init --help`.")
	}

	cfg := aws.NewConfig()
	if cmd.region != "" {
		_, ok := endpoints.AwsPartition().Regions()[cmd.region]
		if !ok {
			return ErrInvalidAWSRegion
		}
		cfg = cfg.WithRegion(cmd.region)
	}

	// Disable retries to make sure we quickly fail if no credentials are present.
	noRetries := aws.NewConfig().WithMaxRetries(0)
	sess, err := session.NewSession(cfg, noRetries)
	if err != nil {
		return handleAWSErr(err)
	}
	stsSvc := sts.New(sess)

	identity, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return handleAWSErr(err)
	}
	accountID := aws.StringValue(identity.Account)

	fmt.Fprintf(cmd.io.Output(), "Detected access to AWS account %s.", accountID)

	if cfg.Region == nil && cmd.kmsKeyID != "" {
		// When the region is not configured in the AWS configuration and not supplied using the flag, use
		// the region from the KMS key if the key is supplied as an ARN.
		kmsARN, err := arn.Parse(cmd.kmsKeyID)
		if err == nil {
			cfg = cfg.WithRegion(kmsARN.Region)
		}
	}

	if cfg.Region != nil {
		fmt.Fprintf(cmd.io.Output(), "Using region %s.", *cfg.Region)
	}
	fmt.Fprintln(cmd.io.Output())

	if cfg.Region == nil {
		region, err := ui.ChooseDynamicOptions(cmd.io, "Which region do you want to use for KMS?", getAWSRegionOptions, true, "region")
		if err != nil {
			return err
		}

		_, ok := endpoints.AwsPartition().Regions()[region]
		if !ok {
			return ErrInvalidAWSRegion
		}
		cfg = cfg.WithRegion(region)
	}

	if cmd.role == "" {
		role, err := ui.AskAndValidate(cmd.io, "What IAM role should have access to the service? (ARN or role name)\n", 3, checkIsNotEmpty("role"))
		if err != nil {
			return err
		}
		cmd.role = role
	}

	if cmd.kmsKeyID == "" {
		kmsKeyOptionsGetter := newKMSKeyOptionsGetter(cfg)
		kmsKey, err := ui.ChooseDynamicOptions(cmd.io, "What is the KMS-key you want to use for encrypting this service's credential? (ARN or ID) The service's IAM role should have decryption permissions on this key.", kmsKeyOptionsGetter.get, true, "KMS key (ARN or ID)")
		if err != nil {
			return err
		}
		cmd.kmsKeyID = kmsKey
	}

	if cmd.description == "" {
		cmd.description = "AWS role " + roleNameFromRole(cmd.role)
	}

	service, err := client.Services().Create(cmd.repo.Value(), cmd.description, credentials.CreateAWS(cmd.kmsKeyID, cmd.role, cfg))
	if err != nil {
		return err
	}

	if cmd.permission != "" {
		err = givePermission(service, cmd.repo, cmd.permission, client)
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.io.Output(), "Successfully created a new service account with ID: "+service.ServiceID)
	fmt.Fprintf(cmd.io.Output(), "Any host that assumes the IAM role %s can now automatically authenticate to SecretHub and fetch the secrets the service has been given access to.\n", roleNameFromRole(cmd.role))

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceAWSInitCommand) Register(r cli.Registerer) {
	clause := r.Command("init", "Create a new service account that is tied to an AWS IAM role.")
	clause.Cmd.Args = cobra.ExactValidArgs(1)
	//clause.Arg("repo", "The service account is attached to the repository in this path.").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.repo)
	clause.StringVar(&cmd.kmsKeyID, "kms-key", "", "The ID or ARN of the KMS-key to be used for encrypting the service's account key.", true, false)
	clause.StringVar(&cmd.role, "role", "", "The role name or ARN of the IAM role that should have access to this service account.", true, false)
	clause.StringVar(&cmd.region, "region", "", "The AWS region that should be used for KMS.", true, false)
	clause.StringVar(&cmd.description, "description", "", "A description for the service so others will recognize it. Defaults to the name of the role that is attached to the service.", true, false)
	clause.StringVar(&cmd.description, "descr", "", "", true, false)
	clause.StringVar(&cmd.description, "desc", "", "", true, false)
	clause.Cmd.Flag("desc").Hidden = true
	clause.Cmd.Flag("descr").Hidden = true
	clause.StringVar(&cmd.permission, "permission", "", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use `--permission <permission>` to give permission on the root of the repo and `--permission <dir>[/<dir> ...]:<permission>` to give permission on a subdirectory.", true, false)

	clause.HelpLong("The native AWS identity provider uses a combination of AWS IAM and AWS KMS to provide access to SecretHub for any service running on AWS (e.g. EC2, Lambda or ECS). For this to work, an IAM role and a KMS key are needed.\n" +
		"\n" +
		"  - The role should be the role that is assumed by the service during execution: the \"IAM Role\" for an EC2 instance, \"Task Execution Role\" for an Lambda Function and \"Task Role\" for ECS Tasks.\n" +
		"  - The KMS key is a key that is used for encryption of the account. Decryption permission on this key must be granted to the previously described IAM Role.\n" +
		"\n" +
		"To create a new service that uses the AWS identity provider, the CLI must have encryption access to the KMS key that will be used by the service account. Therefore AWS credentials should be configured on this system. For details on how this can be done, see https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html.\n" +
		"\n" +
		"If no system-wide default for the AWS region is provided (e.g. with $AWS_REGION), the AWS-region where the KMS key resides should be explicitly provided to this command with the --region flag.",
	)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.repo})
}

func newKMSKeyOptionsGetter(cfg *aws.Config) kmsKeyOptionsGetter {
	return kmsKeyOptionsGetter{
		cfg:           cfg,
		timeFormatter: NewTimeFormatter(false),
	}
}

func getAWSRegionOptions() ([]ui.Option, bool, error) {
	regions := endpoints.AwsPartition().Regions()
	options := make([]ui.Option, len(regions))
	regionNames := make([]string, len(regions))
	i := 0
	for name := range regions {
		regionNames[i] = name
		i++
	}
	sort.Strings(regionNames)

	for i, name := range regionNames {
		region := regions[name]
		options[i] = ui.Option{
			Value:   region.ID(),
			Display: region.ID() + "\t" + region.Description(),
		}
	}
	return options, true, nil
}

type kmsKeyOptionsGetter struct {
	cfg           *aws.Config
	timeFormatter TimeFormatter

	done       bool
	nextMarker string
}

func (g *kmsKeyOptionsGetter) get() ([]ui.Option, bool, error) {
	if g.done {
		return []ui.Option{}, true, nil
	}

	listKeysInput := kms.ListKeysInput{}
	listKeysInput.SetLimit(10)
	if g.nextMarker != "" {
		listKeysInput.SetMarker(g.nextMarker)
	}

	sess, err := session.NewSession(g.cfg)
	if err != nil {
		return nil, true, handleAWSErr(err)
	}
	kmsSvc := kms.New(sess)

	keys, err := kmsSvc.ListKeys(&listKeysInput)
	if err != nil {
		return nil, true, handleAWSErr(err)
	}

	if keys.NextMarker != nil {
		g.nextMarker = *keys.NextMarker
	} else {
		g.done = true
	}

	var waitgroup sync.WaitGroup
	options := make([]*ui.Option, len(keys.Keys))

	for i, key := range keys.Keys {
		waitgroup.Add(1)
		go func(i int, key *kms.KeyListEntry) {
			option := ui.Option{
				Value:   aws.StringValue(key.KeyArn),
				Display: aws.StringValue(key.KeyId),
			}

			enabled := true

			resp, err := kmsSvc.DescribeKey(&kms.DescribeKeyInput{KeyId: key.KeyId})
			if err == nil {
				option.Display += " (created " + g.timeFormatter.Format(*resp.KeyMetadata.CreationDate) + ")\n"
				if aws.StringValue(resp.KeyMetadata.Description) != "" {
					option.Display += "Description: " + aws.StringValue(resp.KeyMetadata.Description) + "\n"
				}
				enabled = aws.BoolValue(resp.KeyMetadata.Enabled)
			}
			if enabled {
				options[i] = &option
			}

			waitgroup.Done()
		}(i, key)
	}
	waitgroup.Wait()

	ret := []ui.Option{}

	for _, option := range options {
		if option != nil {
			ret = append(ret, *option)
		}
	}

	if len(ret) == 0 {
		return g.get()
	}

	return ret, g.done, nil
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

func handleAWSErr(err error) error {
	errAWS, ok := err.(awserr.Error)
	if ok {
		if errAWS.Code() == "NoCredentialProviders" {
			return shaws.ErrNoAWSCredentials
		}
		if errAWS.Code() == "MissingRegion" {
			return ErrMissingRegion
		}
		err = errio.Namespace("aws").Code(errAWS.Code()).Error(errAWS.Message())
	}
	return fmt.Errorf("error fetching available KMS keys: %s", err)
}

func checkIsNotEmpty(name string) func(string) error {
	return func(v string) error {
		if v == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		return nil
	}
}
