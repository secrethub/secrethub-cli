package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/filemode"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/credentials"
	"github.com/spf13/cobra"
)

// ServiceInitCommand initializes a service and writes the generated config to stdout.
type ServiceInitCommand struct {
	clip        bool
	description string
	file        string
	fileMode    filemode.FileMode
	repo        api.RepoPath
	permission  string
	clipper     clip.Clipper
	io          ui.IO
	newClient   newClientFunc
}

// NewServiceInitCommand creates a new ServiceInitCommand.
func NewServiceInitCommand(io ui.IO, newClient newClientFunc) *ServiceInitCommand {
	return &ServiceInitCommand{
		clipper:   clip.NewClipboard(),
		io:        io,
		newClient: newClient,
	}
}

// Run initializes a service and writes the generated config to stdout.
func (cmd *ServiceInitCommand) Run() error {
	var err error

	if cmd.file != "" {
		_, err := os.Stat(cmd.file)
		if !os.IsNotExist(err) {
			return ErrFileAlreadyExists
		}
	}

	if cmd.clip && cmd.file != "" {
		return ErrFlagsConflict("--clip and --file")
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	credential := credentials.CreateKey()
	service, err := client.Services().Create(cmd.repo.Value(), cmd.description, credential)
	if err != nil {
		return err
	}

	if cmd.permission != "" {
		err = givePermission(service, cmd.repo, cmd.permission, client)
		if err != nil {
			return err
		}
	}
	out, err := credential.Export()
	if err != nil {
		return err
	}

	if cmd.clip {
		err = WriteClipboardAutoClear(out, defaultClearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "Copied account configuration for %s to clipboard. It will be cleared after 45 seconds.\n", service.ServiceID)
	} else if cmd.file != "" {
		err = ioutil.WriteFile(cmd.file, posix.AddNewLine(out), cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.file, err)
		}

		fmt.Fprintf(
			cmd.io.Output(),
			"Written account configuration for %s to %s. Be sure to remove it when you're done.\n",
			service.ServiceID,
			cmd.file,
		)
	} else {
		fmt.Fprintf(cmd.io.Output(), "%s", posix.AddNewLine(out))
	}

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceInitCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("init", "Create a new service account.")
	clause.Args = cobra.ExactValidArgs(1)
	//clause.Arg("repo", "The service account is attached to the repository in this path.").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.repo)
	clause.Flag("description", "A description for the service so others will recognize it.").StringVar(&cmd.description)
	clause.Flag("descr", "").Hidden().StringVar(&cmd.description)
	clause.Flag("desc", "").Hidden().StringVar(&cmd.description)
	clause.Flag("permission", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use `--permission <permission>` to give permission on the root of the repo and `--permission <dir>[/<dir> ...]:<permission>` to give permission on a subdirectory.").StringVar(&cmd.permission)
	// TODO make 45 sec configurable
	clause.Flag("clip", "Write the service account configuration to the clipboard instead of stdout. The clipboard is automatically cleared after 45 seconds.").Short('c').BoolVar(&cmd.clip)
	clause.Flag("file", "Write the service account configuration to a file instead of stdout.").Hidden().StringVar(&cmd.file)
	clause.Flag("out-file", "Write the service account configuration to a file instead of stdout.").StringVar(&cmd.file)
	clause.Flag("file-mode", "Set filemode for the written file. Defaults to 0440 (read only) and is ignored without the --file flag.").Default("0440").SetValue(&cmd.fileMode)

	command.BindAction(clause, cmd.PreRun, cmd.Run)
}

// givePermission gives the service permission on the repository as defined in the permission flag.
// When the permission flag is given in the format <permission>, the permission is given on the root directory of the repository.
// When the permission flag is given in the format <subdirectory>:<permission>, the permission is given on the given subdirectory of the
// repo.
func givePermission(service *api.Service, repo api.RepoPath, permissionFlagValue string, client secrethub.ClientInterface) error {
	subdir, permissionValue := parsePermissionFlag(permissionFlagValue)

	permissionPath, err := api.NewDirPath(api.JoinPaths(repo.GetDirPath().String(), subdir))
	if err != nil {
		return ErrInvalidPermissionPath(err)
	}

	var permission api.Permission
	err = permission.Set(permissionValue)
	if err != nil {
		return err
	}

	if permission != 0 {
		_, err := client.AccessRules().Set(permissionPath.Value(), permission.String(), service.ServiceID)
		if err != nil {
			_, delErr := client.Services().Delete(service.ServiceID)
			if delErr != nil {
				fmt.Fprintf(os.Stderr, "Failed to cleanup after creating an access rule for %s failed. Be sure to manually remove the created service account %s: %s\n", service.ServiceID, service.ServiceID, err)
				return delErr
			}

			return err
		}
	}

	return nil
}

func (cmd *ServiceInitCommand) PreRun(c *cobra.Command, args []string) error {
	var err error
	cmd.repo, err = api.NewRepoPath(args[0])
	if err != nil {
		return err
	}
	return nil
}

// parsePermissionFlag parses a permission flag into a permission and a subdirectory to give
// the permission on.
func parsePermissionFlag(value string) (subdir string, permission string) {
	values := strings.SplitN(value, ":", 2)
	if len(values) == 1 {
		return "", values[0]
	}
	return values[0], values[1]
}
