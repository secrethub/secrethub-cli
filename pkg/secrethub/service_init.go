package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/keylockerbv/secrethub-cli/pkg/cli/flags/filemode"
	"github.com/keylockerbv/secrethub-cli/pkg/clip"
	"github.com/keylockerbv/secrethub-cli/pkg/posix"
	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

// ServiceInitCommand initializes a service and writes the generated config to stdout.
type ServiceInitCommand struct {
	clip        bool
	description string
	file        string
	fileMode    filemode.FileMode
	path        api.DirPath
	permission  api.Permission
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
		return errio.Error(err)
	}

	serviceCredential, err := secrethub.GenerateCredential()
	if err != nil {
		return errio.Error(err)
	}

	encoded, err := secrethub.EncodeCredential(serviceCredential)
	if err != nil {
		return errio.Error(err)
	}

	service, err := client.Services().Create(repo.Value(), cmd.description, serviceCredential)
	if err != nil {
		return errio.Error(err)
	}

	if cmd.permission != 0 {
		_, err = client.AccessRules().Set(cmd.path.Value(), cmd.permission, service.ServiceID)
		if err != nil {
			_, delErr := client.Services().Delete(service.ServiceID)
			if delErr != nil {
				fmt.Fprintf(cmd.io.Stdout(), "Failed to cleanup after creating an access rule for %s failed. Be sure to manually remove the created service account %s: %s\n", service.ServiceID, service.ServiceID, err)
				return errio.Error(delErr)
			}

			return errio.Error(err)
		}
	}

	out := []byte(encoded)
	if cmd.clip {
		err = WriteClipboardAutoClear(out, defaultClearClipboardAfter, cmd.clipper)
		if err != nil {
			return errio.Error(err)
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
	clause.Arg("path", "The service account is attached to the repository in this path and when used together with --permission, an access rule is created on the directory in this path.").Required().SetValue(&cmd.path)
	clause.Flag("desc", "A description for the service").StringVar(&cmd.description)
	clause.Flag("permission", "Automatically create an access rule giving the service account permission on the given path argument. Accepts `read`, `write` or `admin`.").SetValue(&cmd.permission)
	// TODO make 45 sec configurable
	clause.Flag("clip", "Write the service account configuration to the clipboard instead of stdout. The clipboard is automatically cleared after 45 seconds.").Short('c').BoolVar(&cmd.clip)
	clause.Flag("file", "Write the service account configuration to a file instead of stdout.").StringVar(&cmd.file)
	clause.Flag("file-mode", "Set filemode for the written file. Defaults to 0440 (read only) and is ignored without the --file flag.").Default("0440").SetValue(&cmd.fileMode)

	BindAction(clause, cmd.Run)
}
