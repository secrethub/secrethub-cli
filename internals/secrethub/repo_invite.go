package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
)

// RepoInviteCommand handles inviting a user to collaborate on a repository.
type RepoInviteCommand struct {
	path      api.RepoPath
	username  string
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
func (cmd *RepoInviteCommand) Register(r Registerer) {
	clause := r.Command("invite", "Invite a user to collaborate on a repository.")
	clause.Arg("repo-path", "The repository to invite the user to (<namespace>/<repo>)").Required().SetValue(&cmd.path)
	clause.Arg("username", "username of the user").Required().StringVar(&cmd.username)
	registerForceFlag(clause).BoolVar(&cmd.force)

	BindAction(clause, cmd.Run)
}

// Run invites the configured user to collaborate on the repo.
func (cmd *RepoInviteCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	user, err := client.Users().Get(cmd.username)
	if err != nil {
		return err
	}

	if !cmd.force {
		msg := fmt.Sprintf("Are you sure you want to add %s to the %s repository?",
			user.PrettyName(),
			cmd.path)

		confirmed, err := ui.AskYesNo(cmd.io, msg, ui.DefaultNo)
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Fprintln(cmd.io.Stdout(), "Aborting.")
			return nil
		}
	}
	fmt.Fprintln(cmd.io.Stdout(), "Inviting user...")

	_, err = client.Repos().Users().Invite(cmd.path.Value(), cmd.username)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "Invite complete! The user %s is now a member of the %s repository.\n", user.Username, cmd.path)

	return nil
}
