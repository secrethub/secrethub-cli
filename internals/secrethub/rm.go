package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/spf13/cobra"
)

// Errors
var (
	ErrCannotRemoveDir     = errMain.Code("cannot_remove_dir").Error("cannot remove directory. Use the -r flag to remove directories.")
	ErrCannotRemoveRootDir = errMain.Code("cannot_remove_root_dir").Errorf(
		"cannot remove root directory. Use the repo rm command to remove a repository",
	)
)

// RmCommand handles removing a resource.
type RmCommand struct {
	path      api.Path
	recursive bool
	force     bool
	io        ui.IO
	newClient newClientFunc
}

// NewRmCommand creates a new RmCommand.
func NewRmCommand(io ui.IO, newClient newClientFunc) *RmCommand {
	return &RmCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RmCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("rm", "Remove a directory, secret or version.")
	clause.Alias("remove")
	clause.Args = cobra.ExactValidArgs(1)
	clause.ValidArgsFunction = AutoCompleter{client: GetClient()}.SecretSuggestions
	//clause.Arg("path", "The path to the resource to remove (<namespace>/<repo>[/<path>])").Required().SetValue(&cmd.path)
	clause.BoolVarP(&cmd.recursive, "recursive", "r", false, "Remove directories and their contents recursively.", true, false)
	registerForceFlag(clause, &cmd.force)

	command.BindAction(clause, cmd.argumentRegister, cmd.Run)
}

// Run removes the resource at the given path.
// Removes a secret, secret-version or directory.
// To remove a directory the -r flag must be set.
func (cmd *RmCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	if !cmd.path.HasVersion() {
		dirPath, err := cmd.path.ToDirPath()
		if err != nil {
			return err
		}

		if dirPath.IsRepoPath() {
			return ErrCannotRemoveRootDir
		}

		_, err = client.Dirs().GetTree(dirPath.Value(), -1, false)
		if err == nil {
			if !cmd.recursive {
				return ErrCannotRemoveDir
			}
			return rmDir(client, dirPath, cmd.force, cmd.io)
		} else if !api.IsErrNotFound(err) {
			return err
		}
	}

	secretPath, err := cmd.path.ToSecretPath()
	if err != nil {
		return err
	}

	if cmd.path.HasVersion() {
		return rmSecretVersion(client, secretPath, cmd.force, cmd.io)
	}

	// Check if the secret exists first so we can return a generic error here instead of ErrSecretNotFound.
	_, err = client.Secrets().Get(secretPath.Value())
	if api.IsErrNotFound(err) {
		return ErrResourceNotFound(cmd.path)
	}

	return rmSecret(client, secretPath, cmd.force, cmd.io)
}

func (cmd *RmCommand) argumentRegister(c *cobra.Command, args []string) error {
	var err error
	cmd.path, err = api.NewPath(args[0])
	if err != nil {
		return err
	}
	return nil
}

func rmSecretVersion(client secrethub.ClientInterface, secretPath api.SecretPath, force bool, io ui.IO) error {
	version, err := secretPath.GetVersion()
	if err != nil {
		return err
	}

	ok, err := askRmConfirmation(
		io,
		fmt.Sprintf("This will permanently remove the %s secret version. "+
			"Please type in the name of the secret and the version (<name>:<version>) to confirm", secretPath.String()),
		force,
		fmt.Sprintf("%s:%s", secretPath.GetSecret(), version),
		secretPath.String(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	err = client.Secrets().Versions().Delete(secretPath.Value())
	if err != nil {
		return err
	}

	fmt.Fprintf(
		io.Output(),
		"Removal complete! The secret version %s has been permanently removed.\n",
		secretPath,
	)

	return nil
}

func rmSecret(client secrethub.ClientInterface, secretPath api.SecretPath, force bool, io ui.IO) error {
	ok, err := askRmConfirmation(
		io,
		fmt.Sprintf("This will permanently remove the %s secret and all its versions. "+
			"Please type in the name of the secret to confirm", secretPath.String()),
		force,
		secretPath.GetSecret(),
		secretPath.String(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	err = client.Secrets().Delete(secretPath.Value())
	if err != nil {
		return err
	}

	fmt.Fprintf(
		io.Output(),
		"Removal complete! The secret %s has been permanently removed.\n",
		secretPath,
	)

	return nil
}

func rmDir(client secrethub.ClientInterface, dirPath api.DirPath, force bool, io ui.IO) error {
	ok, err := askRmConfirmation(
		io,
		fmt.Sprintf("This will permanently remove the %s directory and all the directories and secrets it contains. "+
			"Please type in the name of the directory to confirm", dirPath.String()),
		force,
		dirPath.GetDirName(),
		dirPath.String(),
	)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	err = client.Dirs().Delete(dirPath.Value())
	if err != nil {
		return err
	}

	fmt.Fprintf(
		io.Output(),
		"Removal complete! The directory %s has been permanently removed.\n",
		dirPath,
	)

	return nil
}

func askRmConfirmation(io ui.IO, confirmationText string, force bool, expected ...string) (bool, error) {
	if force {
		return true, nil
	}

	confirmed, err := ui.ConfirmCaseInsensitive(
		io,
		fmt.Sprintf(
			"[WARNING] This action cannot be undone. %s",
			confirmationText,
		),
		expected...,
	)

	if err == ui.ErrCannotAsk {
		return false, ErrCannotDoWithoutForce
	} else if err != nil {
		return false, err
	}

	if !confirmed {
		fmt.Fprintln(io.Output(), "Name does not match. Aborting.")
		return false, nil
	}
	return true, nil
}
