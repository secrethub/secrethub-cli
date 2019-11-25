package secrethub

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-go/internals/api"
)

// AuditCommand is a command to audit a repo or a secret.
type AuditCommand struct {
	io            ui.IO
	path          api.Path
	useTimestamps bool
	timeFormatter TimeFormatter
	newClient     newClientFunc
	perPage       int
}

// NewAuditCommand creates a new audit command.
func NewAuditCommand(io ui.IO, newClient newClientFunc) *AuditCommand {
	return &AuditCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AuditCommand) Register(r command.Registerer) {
	clause := r.Command("audit", "Show the audit log.")
	clause.Arg("repo-path or secret-path", "Path to the repository or the secret to audit "+repoPathPlaceHolder+" or "+secretPathPlaceHolder).SetValue(&cmd.path)
	clause.Flag("per-page", "number of audit events shown per page").Default("20").IntVar(&cmd.perPage)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// Run prints all audit events for the given repository or secret.
func (cmd *AuditCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *AuditCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

// Run prints all audit events for the given repository or secret.
func (cmd *AuditCommand) run() error {
	if cmd.perPage < 1 {
		return fmt.Errorf("per-page should be positive, got %d", cmd.perPage)
	}

	var auditTable auditTable
	var iter secrethub.AuditEventIterator

	repoPath, err := cmd.path.ToRepoPath()
	if err == nil {
		client, err := cmd.newClient()
		if err != nil {
			return err
		}
		tree, err := client.Dirs().GetTree(repoPath.GetDirPath().Value(), -1, false)
		if err != nil {
			return err
		}

		iter = client.Repos().EventIterator(repoPath.Value(), nil)
		auditTable = newRepoAuditTable(tree, cmd.timeFormatter)
	} else {
		secretPath, err := cmd.path.ToSecretPath()
		if err == nil {
			if cmd.path.HasVersion() {
				return ErrCannotAuditSecretVersion
			}

			client, err := cmd.newClient()
			if err != nil {
				return err
			}

			iter = client.Secrets().EventIterator(secretPath.Value(), nil)
			auditTable = newSecretAuditTable(cmd.timeFormatter)
		} else {
			return ErrNoValidRepoOrSecretPath
		}
	}

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)
	header := strings.Join(auditTable.header(), "\t") + "\n"
	fmt.Fprint(tabWriter, header)

	// interactive mode is assumed, except when output is piped.
	interactive := !cmd.io.Stdout().IsPiped()

	i := 0
	for {
		i++
		event, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err == api.ErrSecretNotFound {
			// Check if we're attempting to audit a dir.
			client, err := cmd.newClient()
			if err != nil {
				return err
			}

			_, err = client.Dirs().GetTree(api.DirPath(cmd.path).Value(), 1, false)
			if err == nil {
				return ErrCannotAuditDir
			}
			return api.ErrSecretNotFound
		} else if err != nil {
			return err
		}

		row, err := auditTable.row(event)
		if err != nil {
			return err
		}

		fmt.Fprint(tabWriter, strings.Join(row, "\t")+"\n")

		if interactive && i == cmd.perPage {
			err = tabWriter.Flush()
			if err != nil {
				return err
			}
			i = 0
			fmt.Fprintln(cmd.io.Stdout(), "Press <ENTER> to show more results. Press <CTRL+C> to exit.")

			// wait for <ENTER> to continue.
			_, err := ui.Readln(cmd.io.Stdin())
			if err != nil {
				return err
			}
			fmt.Fprint(tabWriter, header)
		}
	}

	err = tabWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

type auditTable interface {
	header() []string
	row(event api.Audit) ([]string, error)
}

func newBaseAuditTable(timeFormatter TimeFormatter) baseAuditTable {
	return baseAuditTable{
		timeFormatter: timeFormatter,
	}
}

type baseAuditTable struct {
	timeFormatter TimeFormatter
}

func (table baseAuditTable) header(content ...string) []string {
	res := append([]string{"AUTHOR", "EVENT"}, content...)
	return append(res, "IP ADDRESS", "DATE")
}

func (table baseAuditTable) row(event api.Audit, content ...string) ([]string, error) {
	actor, err := getAuditActor(event)
	if err != nil {
		return nil, err
	}

	res := append([]string{actor, getEventAction(event)}, content...)
	return append(res, event.IPAddress, table.timeFormatter.Format(event.LoggedAt)), nil
}

func newSecretAuditTable(timeFormatter TimeFormatter) secretAuditTable {
	return secretAuditTable{
		baseAuditTable: newBaseAuditTable(timeFormatter),
	}
}

type secretAuditTable struct {
	baseAuditTable
}

func (table secretAuditTable) header() []string {
	return table.baseAuditTable.header()
}

func (table secretAuditTable) row(event api.Audit) ([]string, error) {
	return table.baseAuditTable.row(event)
}

func newRepoAuditTable(tree *api.Tree, timeFormatter TimeFormatter) repoAuditTable {
	return repoAuditTable{
		baseAuditTable: newBaseAuditTable(timeFormatter),
		tree:           tree,
	}
}

type repoAuditTable struct {
	baseAuditTable
	tree *api.Tree
}

func (table repoAuditTable) header() []string {
	return table.baseAuditTable.header("EVENT SUBJECT")
}

func (table repoAuditTable) row(event api.Audit) ([]string, error) {
	subject, err := getAuditSubject(event, table.tree)
	if err != nil {
		return nil, err
	}

	return table.baseAuditTable.row(event, subject)
}
