package secrethub

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/spf13/cobra"
)

// RepoRevokeCommand handles revoking an account access to a repository.
type RepoRevokeCommand struct {
	accountName api.AccountName
	path        api.RepoPath
	force       bool
	io          ui.IO
	newClient   newClientFunc
}

// NewRepoRevokeCommand creates a new RepoRevokeCommand.
func NewRepoRevokeCommand(io ui.IO, newClient newClientFunc) *RepoRevokeCommand {
	return &RepoRevokeCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RepoRevokeCommand) Register(r cli.Registerer) {
	clause := r.Command("revoke", "Revoke an account's access to a repository. A list of secrets that should be rotated will be printed out.")
	clause.Cmd.Args = cobra.ExactValidArgs(2)
	//clause.Arg("repo-path", "The repository to revoke the account from").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.path)
	//clause.Arg("account-name", "The account name (username or service name) to revoke access for").Required().SetValue(&cmd.accountName)
	registerForceFlag(clause, &cmd.force)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.path, &cmd.accountName})
}

// Run removes and revokes access to an account from a repo if possible.
func (cmd *RepoRevokeCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	var prettyName string
	if cmd.accountName.IsUser() {
		user, err := client.Users().Get(string(cmd.accountName))
		if err != nil {
			return err
		}
		prettyName = user.PrettyName()
	} else {
		prettyName = string(cmd.accountName)
	}

	if !cmd.force {
		msg := fmt.Sprintf("Are you sure you want to revoke %s from the repository %s?",
			prettyName,
			cmd.path,
		)

		confirmed, err := ui.AskYesNo(cmd.io, msg, ui.DefaultNo)
		if err == ui.ErrCannotAsk {
			return ErrCannotDoWithoutForce
		} else if err != nil {
			return err
		}

		if !confirmed {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}
	}

	fmt.Fprint(cmd.io.Output(), "Revoking account...\n\n")

	var revoked *api.RevokeRepoResponse
	if cmd.accountName.IsService() {
		revoked, err = client.Services().Delete(string(cmd.accountName))
	} else {
		revoked, err = client.Repos().Users().Revoke(cmd.path.Value(), string(cmd.accountName))
	}
	if err != nil {
		return err
	}

	if revoked.Status == api.StatusFailed {
		fmt.Fprintf(cmd.io.Output(),
			"\nRevoke failed! The account %s is the only admin on the repo %s."+
				"You need to make sure another account has admin rights on the repository or you can remove the repo.",
			prettyName,
			cmd.path,
		)
	}

	rootDir, err := client.Dirs().GetTree(cmd.path.GetDirPath().Value(), -1, false)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)

	countUnaffected, countFlagged := printFlaggedSecrets(w, rootDir.RootDir, cmd.path.GetNamespace())

	err = w.Flush()
	if err != nil {
		return err
	}

	if countFlagged > 0 {
		fmt.Fprintln(cmd.io.Output())
	}
	fmt.Fprintf(cmd.io.Output(),
		"Revoke complete! The account %s can no longer access the %s repository. "+
			"Make sure you overwrite or delete all flagged secrets. "+
			"Secrets: %d unaffected, %d flagged\n",
		prettyName,
		cmd.path,
		countUnaffected,
		countFlagged,
	)

	return nil
}

func printFlaggedSecrets(w io.Writer, dir *api.Dir, prePath string) (int, int) {
	var countUnaffected, countFlagged int
	if prePath != "" {
		prePath = fmt.Sprintf("%s/%s", prePath, dir.Name)
	} else {
		prePath = dir.Name
	}

	// Print the directories below
	for _, subDir := range dir.SubDirs {
		subUnaffected, subFlagged := printFlaggedSecrets(w, subDir, prePath)
		countUnaffected += subUnaffected
		countFlagged += subFlagged
	}

	// Print the secrets below
	for _, secret := range dir.Secrets {
		if secret.Status != api.StatusOK {
			countFlagged++
			fmt.Fprintf(w, "%s/%s\t=> %s\n", prePath, secret.Name, secret.Status)
		} else {
			countUnaffected++
		}
	}

	return countUnaffected, countFlagged
}
