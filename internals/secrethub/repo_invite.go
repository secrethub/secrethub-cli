package secrethub

import (
	"fmt"
	"github.com/spf13/cobra"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	// "github.com/spf13/cobra"
)

// RepoInviteCommand handles inviting a user to collaborate on a repository.
type RepoInviteCommand struct {
	path      api.RepoPath
	username  cli.StringArgValue
	force     bool
	io        ui.IO
	newClient newClientFunc
}

// NewRepoInviteCommand creates a new RepoInviteCommand.
func NewRepoInviteCommand(io ui.IO, newClient newClientFunc) *RepoInviteCommand {
	return &RepoInviteCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RepoInviteCommand) Register(r cli.Registerer) {
	clause := r.Command("invite", "Invite a user to collaborate on a repository.")
	clause.Cmd.Args = cobra.MaximumNArgs(2)
	//clause.Arg("repo-path", "The repository to invite the user to").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.path)
	//clause.Arg("username", "username of the user").Required().StringVar(&cmd.username)
	registerForceFlag(clause, &cmd.force)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.path, &cmd.username}, []string{"repo-path", "username"})
}

// Run invites the configured user to collaborate on the repo.
func (cmd *RepoInviteCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	if !cmd.force {
		user, err := client.Users().Get(cmd.username.Param)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("Are you sure you want to add %s to the %s repository?",
			user.PrettyName(),
			cmd.path)

		confirmed, err := ui.AskYesNo(cmd.io, msg, ui.DefaultNo)
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}
	}
	fmt.Fprintln(cmd.io.Output(), "Inviting user...")

	_, err = client.Repos().Users().Invite(cmd.path.Value(), cmd.username.Param)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Output(), "Invite complete! The user %s is now a member of the %s repository.\n", cmd.username.Param, cmd.path)

	return nil
}
