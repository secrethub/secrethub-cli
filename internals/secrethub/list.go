package secrethub

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// LsCommand lists a repo, secret or namespace.
type LsCommand struct {
	path          api.Path
	quiet         bool
	useTimestamps bool
	io            ui.IO
	newClient     newClientFunc
}

// NewLsCommand creates a new LsCommand.
func NewLsCommand(io ui.IO, newClient newClientFunc) *LsCommand {
	return &LsCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *LsCommand) Register(r command.Registerer) {
	clause := r.Command("ls", "List contents of a path.")
	clause.Alias("list")
	clause.Arg("path", "The path to list contents of").PlaceHolder(anyPathPlaceHolder).SetValue(&cmd.path)
	clause.Flag("quiet", "Only print paths.").Short('q').BoolVar(&cmd.quiet)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// Run lists a repo, secret or namespace.
func (cmd *LsCommand) Run() error {
	timeFormatter := NewTimeFormatter(cmd.useTimestamps)

	if cmd.path == "" {
		repoLSCommand := NewRepoLSCommand(cmd.io, cmd.newClient)
		repoLSCommand.quiet = cmd.quiet
		repoLSCommand.useTimestamps = cmd.useTimestamps
		return repoLSCommand.Run()
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	// It must be a SecretPath as only SecretPaths has versions.
	if cmd.path.HasVersion() {
		secretPath, err := cmd.path.ToSecretPath()
		if err != nil {
			fmt.Println("no secret path!")
			return err
		}

		version, err := client.Secrets().Versions().GetWithoutData(secretPath.Value())
		if err != nil {
			return err
		}

		err = printVersions(cmd.io.Stdout(), cmd.quiet, timeFormatter, version)
		if err != nil {
			return err
		}

		return nil
	}

	// Try DirPath
	dirPath, err := cmd.path.ToDirPath()
	if err == nil {
		dirFS, err := client.Dirs().GetTree(dirPath.Value(), 1, false)
		if err == api.ErrDirNotFound && dirPath.IsRepoPath() {
			return err
		} else if err != nil && err != api.ErrDirNotFound {
			return err
		} else if err == nil {
			err = printDir(cmd.io.Stdout(), cmd.quiet, dirFS.RootDir, timeFormatter)
			if err != nil {
				return err
			}
			return nil
		}
	}

	// Try SecretPath
	secretPath, err := cmd.path.ToSecretPath()
	if err == nil {
		versions, err := client.Secrets().Versions().ListWithoutData(secretPath.Value())
		if err == api.ErrSecretNotFound {
			return ErrResourceNotFound(cmd.path)
		} else if err != nil {
			return err
		}

		err = printVersions(cmd.io.Stdout(), cmd.quiet, timeFormatter, versions...)
		if err != nil {
			return err
		}

		return nil
	}

	workspace, err := cmd.path.ToNamespace()
	if err == nil {
		cmd := RepoLSCommand{
			workspace:     workspace,
			useTimestamps: cmd.useTimestamps,
			quiet:         cmd.quiet,
			io:            cmd.io,
			newClient:     cmd.newClient,
		}

		return cmd.Run()
	}

	// Path should always be a namespace, repository, directory, secret or secret version.
	// Therefore, this should never happen.
	return errio.UnexpectedError(errors.New("invalid path argument"))
}

// printVersions prints out secret versions in long or short format.
func printVersions(w io.Writer, quiet bool, timeFormatter TimeFormatter, versions ...*api.SecretVersion) error {
	if quiet {
		for _, version := range versions {
			fmt.Fprintf(w, "%s\n", version.Name())
		}
	} else {
		w := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintf(w, "%s\t%s\t%s\n", "NAME", "STATUS", "CREATED")
		for _, version := range versions {
			fmt.Fprintf(w, "%s\t%s\t%s\n", version.Name(), version.Status, timeFormatter.Format(version.CreatedAt.Local()))
		}
		err := w.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

// printDir prints out directory contents in long or short format.
func printDir(w io.Writer, quiet bool, dir *api.Dir, timeFormatter TimeFormatter) error {
	sort.Sort(api.SortDirByName(dir.SubDirs))
	sort.Sort(api.SortSecretByName(dir.Secrets))

	if quiet {
		for _, dir := range dir.SubDirs {
			fmt.Fprintf(w, "%s/\n", dir.Name)
		}
		for _, secret := range dir.Secrets {
			fmt.Fprintf(w, "%s\n", secret.Name)
		}
	} else {
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintf(tw, "%s\t%s\t%s\n", "NAME", "STATUS", "CREATED")
		for _, dir := range dir.SubDirs {
			fmt.Fprintf(tw, "%s/\t%s\t%s\n", dir.Name, dir.Status, timeFormatter.Format(dir.CreatedAt.Local()))
		}
		for _, secret := range dir.Secrets {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", secret.Name, secret.Status, timeFormatter.Format(secret.CreatedAt.Local()))
		}
		err := tw.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}
