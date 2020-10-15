package secrethub

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/posix"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/spf13/cobra"
)

// Error
var (
	ErrExportAlreadyExists = errMain.Code("export_file_already_exists").Error("the export file already exists")
)

// RepoExportCommand exports a repo to a zip file.
type RepoExportCommand struct {
	path      api.RepoPath
	zipName   cli.StringArgValue
	io        ui.IO
	newClient newClientFunc
}

// NewRepoExportCommand creates a new RepoExportCommand.
func NewRepoExportCommand(io ui.IO, newClient newClientFunc) *RepoExportCommand {
	return &RepoExportCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *RepoExportCommand) Register(r cli.Registerer) {
	clause := r.Command("export", "Export the repository to a zip file.")
	clause.Cmd.Args = cobra.MaximumNArgs(2)
	//clause.Arg("repo-path", "The repository to export").Required().PlaceHolder(repoPathPlaceHolder).SetValue(&cmd.path)
	//clause.Arg("zip-file-name", "The file name to assign to the exported .zip file. Defaults to secrethub_export_<namespace>_<repo>_<timestamp>.zip with the timestamp formatted as YYYYMMDD_HHMMSS").StringVar(&cmd.zipName)

	clause.BindAction(cmd.Run)
	clause.BindArguments([]cli.ArgValue{&cmd.path, &cmd.zipName}, []string{"repo-path"})
}

// Run exports a repo to a zip file
func (cmd *RepoExportCommand) Run() error {
	if cmd.zipName.Param == "" {
		// secrethub_export_repo_date_time.zip
		cmd.zipName.Param = fmt.Sprintf("%s_export_%s_%s.zip", ApplicationName, cmd.path.GetRepo(), time.Now().Format("20060102_150405"))
	}

	_, err := os.Stat(cmd.zipName.Param)
	if err == nil {
		return ErrExportAlreadyExists
	}

	confirmed, err := ui.ConfirmCaseInsensitive(
		cmd.io,
		fmt.Sprintf(
			"[DANGER ZONE] This will export all the secrets unencrypted in the %s repository. "+
				"You are responsible for the protection of these secrets. "+
				"Please type in the full path of the repository to confirm",
			cmd.path.String(),
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

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	rootDir, err := client.Dirs().GetTree(cmd.path.GetDirPath().Value(), -1, false)
	if err != nil {
		return err
	}

	zipFile, err := os.Create(cmd.zipName.Param)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(zipFile)
	defer func() {
		err := writer.Close()
		if err != nil {
			panic(fmt.Errorf("could not close zip file: %s", err))
		}

		err = zipFile.Close()
		if err != nil {
			panic(fmt.Errorf("could not close zip file: %s", err))
		}
	}()

	for _, secret := range rootDir.Secrets {
		secretPath, err := rootDir.AbsSecretPath(secret.SecretID)
		if err != nil {
			return err
		}

		versions, err := client.Secrets().Versions().ListWithData(secretPath.Value())
		if err != nil {
			return err
		}

		for _, version := range versions {
			versionPath, err := secretPath.AddVersion(version.Version)
			if err != nil {
				return err
			}

			// Replace the : for / to create a directory for every secret containing versions.
			zipSecretPath := strings.Replace(versionPath.String(), ":", "/", -1)
			// Remove the repo path from the zipfile.
			zipSecretPath = strings.TrimPrefix(zipSecretPath, versionPath.GetRepoPath().String()+"/")

			zipNode, err := writer.Create(zipSecretPath)
			if err != nil {
				return err
			}

			_, err = zipNode.Write(posix.AddNewLine(version.Data))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
