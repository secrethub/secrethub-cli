package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
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
func (cmd *RepoInspectCommand) Register(r Registerer) {
	clause := r.Command("inspect", "Show the details of a repository.")
	clause.Arg("repo-path", "Path to the repository (<namespace>/<repo>)").Required().SetValue(&cmd.path)

	BindAction(clause, cmd.Run)
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

	fmt.Fprintln(cmd.io.Stdout(), output)

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
