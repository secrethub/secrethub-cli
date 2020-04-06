package secrethub

import (
	"fmt"
	"io"
	"os"

	"github.com/secrethub/secrethub-go/internals/errio"

	"github.com/secrethub/secrethub-cli/internals/secrethub/pager"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/pkg/secrethub"

	"github.com/secrethub/secrethub-go/internals/api"
)

var (
	errAudit        = errio.Namespace("audit")
	errNoSuchFormat = errAudit.Code("invalid_format").ErrorPref("invalid format: %s")
)

const (
	defaultTerminalWidth = 80
	formatTable          = "table"
	formatJSON           = "json"
)

// AuditCommand is a command to audit a repo or a secret.
type AuditCommand struct {
	io                 ui.IO
	newPaginatedWriter func(io.Writer) (io.WriteCloser, error)
	path               api.Path
	useTimestamps      bool
	timeFormatter      TimeFormatter
	newClient          newClientFunc
	terminalWidth      func(int) (int, error)
	perPage            int
	format             string
}

// NewAuditCommand creates a new audit command.
func NewAuditCommand(io ui.IO, newClient newClientFunc) *AuditCommand {
	return &AuditCommand{
		io:                 io,
		newPaginatedWriter: pager.New,
		newClient:          newClient,
		terminalWidth: func(fd int) (int, error) {
			w, _, err := terminal.GetSize(fd)
			return w, err
		},
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AuditCommand) Register(r command.Registerer) {
	clause := r.Command("audit", "Show the audit log.")
	clause.Arg("repo-path or secret-path", "Path to the repository or the secret to audit "+repoPathPlaceHolder+" or "+secretPathPlaceHolder).SetValue(&cmd.path)
	clause.Flag("per-page", "Number of audit events shown per page").Default("20").Hidden().IntVar(&cmd.perPage)
	clause.Flag("output-format", "Specify the format in which to output the log. Options are: table and json. If the output of the command is parsed by a script an alternative of the table format must be used.").HintOptions("table", "json").Default("table").StringVar(&cmd.format)
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

	iter, auditTable, err := cmd.iterAndAuditTable()
	if err != nil {
		return err
	}

	paginatedWriter, err := cmd.newPaginatedWriter(os.Stdout)
	if err == pager.ErrPagerNotFound {
		paginatedWriter = pager.NewFallbackPager(os.Stdout)
	} else if err != nil {
		return err
	}
	defer paginatedWriter.Close()

	var formatter listFormatter
	if cmd.format == formatJSON {
		formatter = newJSONFormatter(paginatedWriter, auditTable.header())
	} else if cmd.format == formatTable {
		terminalWidth, err := cmd.terminalWidth(int(os.Stdout.Fd()))
		if err != nil {
			terminalWidth = defaultTerminalWidth
		}
		formatter = newTableFormatter(paginatedWriter, terminalWidth, auditTable.columns())
	} else {
		return errNoSuchFormat(cmd.format)
	}

	for {
		event, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		row, err := auditTable.row(event)
		if err != nil {
			return err
		}

		err = formatter.Write(row)
		if err == pager.ErrPagerClosed {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *AuditCommand) iterAndAuditTable() (secrethub.AuditEventIterator, auditTable, error) {
	repoPath, err := cmd.path.ToRepoPath()
	if err == nil {
		client, err := cmd.newClient()
		if err != nil {
			return nil, nil, err
		}
		tree, err := client.Dirs().GetTree(repoPath.GetDirPath().Value(), -1, false)
		if err != nil {
			return nil, nil, err
		}

		iter := client.Repos().EventIterator(repoPath.Value(), &secrethub.AuditEventIteratorParams{})
		auditTable := newRepoAuditTable(tree, cmd.timeFormatter)
		return iter, auditTable, nil

	}

	secretPath, err := cmd.path.ToSecretPath()
	if err == nil {
		if cmd.path.HasVersion() {
			return nil, nil, ErrCannotAuditSecretVersion
		}

		client, err := cmd.newClient()
		if err != nil {
			return nil, nil, err
		}

		isDir, err := client.Dirs().Exists(secretPath.Value())
		if err == nil && isDir {
			return nil, nil, ErrCannotAuditDir
		}

		iter := client.Secrets().EventIterator(secretPath.Value(), &secrethub.AuditEventIteratorParams{})
		auditTable := newSecretAuditTable(cmd.timeFormatter)
		return iter, auditTable, nil
	}

	return nil, nil, ErrNoValidRepoOrSecretPath
}

type tableColumn struct {
	name     string
	maxWidth int
}

type auditTable interface {
	header() []string
	row(event api.Audit) ([]string, error)
	columns() []tableColumn
}

func newBaseAuditTable(timeFormatter TimeFormatter, midColumns ...tableColumn) baseAuditTable {
	columns := append([]tableColumn{
		{name: "author", maxWidth: 32},
		{name: "event", maxWidth: 22},
	}, midColumns...)
	columns = append(columns, []tableColumn{
		{name: "IP address", maxWidth: 45},
		{name: "date", maxWidth: 22},
	}...)

	return baseAuditTable{
		tableColumns:  columns,
		timeFormatter: timeFormatter,
	}
}

type baseAuditTable struct {
	tableColumns  []tableColumn
	timeFormatter TimeFormatter
}

func (table baseAuditTable) header() []string {
	res := make([]string, len(table.tableColumns))
	for i, col := range table.tableColumns {
		res[i] = col.name
	}
	return res
}

func (table baseAuditTable) row(event api.Audit, content ...string) ([]string, error) {
	actor, err := getAuditActor(event)
	if err != nil {
		return nil, err
	}

	res := append([]string{actor, getEventAction(event)}, content...)
	return append(res, event.IPAddress, table.timeFormatter.Format(event.LoggedAt)), nil
}

func (table baseAuditTable) columns() []tableColumn {
	return table.tableColumns
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
		baseAuditTable: newBaseAuditTable(timeFormatter, tableColumn{name: "EVENT SUBJECT"}),
		tree:           tree,
	}
}

type repoAuditTable struct {
	baseAuditTable
	tree *api.Tree
}

func (table repoAuditTable) row(event api.Audit) ([]string, error) {
	subject, err := getAuditSubject(event, table.tree)
	if err != nil {
		return nil, err
	}

	return table.baseAuditTable.row(event, subject)
}
