package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/spf13/cobra"
)

// RepoInspectCommand handles printing out the details of a repo in a JSON format.
type RepoInspectCommand struct {
	path          api.RepoPath
	timeFormatter TimeFormatter
	io            ui.IO
	newClient     newClientFunc
}

// NewRepoInspectCommand creates a new RepoInspectCommand.
func NewRepoInspectCommand(io ui.IO, newClient newClientFunc) *RepoInspectCommand {
	return &RepoInspectCommand{
		io:            io,
		newClient:     newClient,
		timeFormatter: NewTimeFormatter(true),
	}
}

// Register registers the command, args, and flags on the provided registerer.
func (cmd *RepoInspectCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("inspect", "Show the details of a repository.")
	clause.Args = cobra.ExactValidArgs(1)
	//clause.Arg("repo-path", "Path to the repository").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.path)

	command.BindAction(clause, cmd.argumentRegister, cmd.Run)
}

// Run prints out the details of a repo.
func (cmd *RepoInspectCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	repo, err := client.Repos().Get(cmd.path.Value())
	if err != nil {
		return err
	}

	users, err := client.Repos().Users().List(cmd.path.Value())
	if err != nil {
		return err
	}

	services, err := client.Repos().Services().List(cmd.path.Value())
	if err != nil {
		return err
	}

	output, err := cli.PrettyJSON(newInspectRepoOutput(repo, users, services, cmd.timeFormatter))
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), output)

	return nil
}

func (cmd *RepoInspectCommand) argumentRegister(c *cobra.Command, args []string) error {
	var err error
	cmd.path, err = api.NewRepoPath(args[0])
	if err != nil {
		return err
	}
	return nil
}

func newInspectRepoOutput(repo *api.Repo, users []*api.User, services []*api.Service, timeFormatter TimeFormatter) inspectRepoOutput {
	out := inspectRepoOutput{
		Name:         repo.Name,
		Owner:        repo.Owner,
		CreatedAt:    timeFormatter.Format(repo.CreatedAt.Local()),
		SecretCount:  repo.SecretCount,
		MemberCount:  len(users),
		ServiceCount: len(services),
		Users:        make([]inspectRepoUserOutput, len(users)),
		Services:     make([]inspectRepoServiceOutput, len(services)),
	}

	for i, user := range users {
		out.Users[i] = newInspectRepoUser(user)
	}

	for i, service := range services {
		out.Services[i] = newInspectRepoService(service)
	}

	return out
}

// inspectRepoOutput is the json format to print out with all the details of a repo.
type inspectRepoOutput struct {
	Name         string
	Owner        string
	CreatedAt    string
	SecretCount  int
	MemberCount  int
	Users        []inspectRepoUserOutput
	ServiceCount int
	Services     []inspectRepoServiceOutput
}

func newInspectRepoUser(user *api.User) inspectRepoUserOutput {
	return inspectRepoUserOutput{
		User:     user.FullName,
		UserName: user.Username,
	}
}

// inspectRepoUserOutput is the json format to print out with all the details of repo users.
type inspectRepoUserOutput struct {
	User     string
	UserName string
}

func newInspectRepoService(service *api.Service) inspectRepoServiceOutput {
	return inspectRepoServiceOutput{
		Service:            service.ServiceID,
		ServiceDescription: service.Description,
	}
}

// inspectRepoServiceOutput is the json format to print out with all the details of repo services.
type inspectRepoServiceOutput struct {
	Service            string
	ServiceDescription string
}
