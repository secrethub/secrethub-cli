package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-cli/internals/cli"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"

	"github.com/spf13/cobra"
)

// RepoRmCommand handles removing a repo.
type RepoRmCommand struct {
	path      api.RepoPath
	io        ui.IO
	newClient newClientFunc
}

// NewRepoRmCommand creates a new RepoRmCommand.
func NewRepoRmCommand(io ui.IO, newClient newClientFunc) *RepoRmCommand {
	return &RepoRmCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RepoRmCommand) Register(r command.Registerer) {
	clause := r.Command("rm", "Permanently delete a repository.")
	clause.Alias("remove")
	clause.Cmd.Args = cobra.ExactValidArgs(1)
	//clause.Arg("repo-path", "The repository to delete").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.path)

	command.BindAction(clause, []cli.ArgValue{&cmd.path}, cmd.Run)
}

// Run removes the repository.
func (cmd *RepoRmCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	_, err = client.Repos().Get(cmd.path.Value())
	if err != nil {
		return err
	}

	confirmed, err := ui.ConfirmCaseInsensitive(
		cmd.io,
		fmt.Sprintf(
			"[DANGER ZONE] This action cannot be undone. "+
				"This will permanently remove the %s repository, all its secrets and all associated service accounts. "+
				"Please type in the full path of the repository to confirm",
			cmd.path,
		),
		cmd.path.String(),
	)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Fprintln(cmd.io.Output(), "Name does not match. Aborting.")
		return nil
	}

	fmt.Fprintln(cmd.io.Output(), "Removing repository...")

	err = client.Repos().Delete(cmd.path.Value())
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Removal complete! The repository %s has been permanently removed.\n", cmd.path)

	return nil
}
