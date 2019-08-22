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

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// ServiceInitCommand initializes a service and writes the generated config to stdout.
type ServiceInitCommand struct {
	clip        bool
	description string
	file        string
	fileMode    filemode.FileMode
	path        api.DirPath
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

	repo := cmd.path.GetRepoPath()

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	serviceCredential, err := secrethub.GenerateCredential()
	if err != nil {
		return err
	}

	encoded, err := secrethub.EncodeCredential(serviceCredential)
	if err != nil {
		return err
	}

	service, err := client.Services().Create(repo.Value(), cmd.description, serviceCredential, serviceCredential)
	if err != nil {
		return err
	}

	if strings.Contains(cmd.permission, ":") && !cmd.path.IsRepoPath() {
		return api.ErrInvalidRepoPath(cmd.path)
	}

	err = givePermission(service, cmd.path.GetRepoPath(), cmd.permission, client)
	if err != nil {
		return err
	}

	out := []byte(encoded)
	if cmd.clip {
		err = WriteClipboardAutoClear(out, defaultClearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Stdout(), "Copied account configuration for %s to clipboard. It will be cleared after 45 seconds.\n", service.ServiceID)
	} else if cmd.file != "" {
		err = ioutil.WriteFile(cmd.file, posix.AddNewLine(out), cmd.fileMode.FileMode())
		if err != nil {
			return ErrCannotWrite(cmd.file, err)
		}

		fmt.Fprintf(
			cmd.io.Stdout(),
			"Written account configuration for %s to %s. Be sure to remove it when you're done.\n",
			service.ServiceID,
			cmd.file,
		)
	} else {
		fmt.Fprintf(cmd.io.Stdout(), "%s", posix.AddNewLine(out))
	}

	return nil
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceInitCommand) Register(r Registerer) {
	clause := r.Command("init", "Create a new service account attached to a repository.")
	clause.Arg("repo", "The service account is attached to the repository in this path.").Required().SetValue(&cmd.path)
	clause.Flag("desc", "A description for the service").StringVar(&cmd.description)
	clause.Flag("permission", "Create an access rule giving the service account permission on a directory. Accepted permissions are `read`, `write` and `admin`. Use <subdirectory>:<permission> format to give permission on a subdirectory of the repo.").StringVar(&cmd.permission)
	// TODO make 45 sec configurable
	clause.Flag("clip", "Write the service account configuration to the clipboard instead of stdout. The clipboard is automatically cleared after 45 seconds.").Short('c').BoolVar(&cmd.clip)
	clause.Flag("file", "Write the service account configuration to a file instead of stdout.").StringVar(&cmd.file)
	clause.Flag("file-mode", "Set filemode for the written file. Defaults to 0440 (read only) and is ignored without the --file flag.").Default("0440").SetValue(&cmd.fileMode)

	BindAction(clause, cmd.Run)
}

func givePermission(service *api.Service, repo api.RepoPath, permissionFlagValue string, client secrethub.Client) error {
	permissionPath := repo.GetDirPath()
	var permission api.Permission
	values := strings.SplitN(permissionFlagValue, ":", 2)
	if len(values) == 1 {
		err := permission.Set(values[0])
		if err != nil {
			return err
		}
	} else if len(values) == 2 {
		err := permission.Set(values[1])
		if err != nil {
			return err
		}

		permissionPath, err = api.NewDirPath(api.JoinPaths(permissionPath.String(), values[0]))
		if err != nil {
			return ErrInvalidPermissionPath(err)
		}
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
