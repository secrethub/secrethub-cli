package secrethub

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/pkg/secrethub"

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

	iter, auditTable, err := cmd.iterAndAuditTable()
	if err != nil {
		return err
	}

	paginatedWriter, done, err := paginateWriter(os.Stdout)
	if err != nil {
		return err
	}

	header := strings.Join(auditTable.header(), "\t") + "\n"
	fmt.Fprint(paginatedWriter, header)

	i := 0
	paginatorClosed := false
	for {
		i++
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

		select {
		case <-done:
			paginatorClosed = true
		default:
			fmt.Fprint(paginatedWriter, strings.Join(row, "\t")+"\n")
		}
		if paginatorClosed {
			break
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

func paginateWriter(outputWriter io.Writer) (io.WriteCloser, <-chan struct{}, error) {
	pager, err := pagerCommand()
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(pager)

	writer, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	cmd.Stdout = outputWriter
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}
	done := make(chan struct{}, 1)
	go func() {
		cmd.Wait()
		done <- struct{}{}
	}()
	return writer, done, nil
}

func pagerCommand() (string, error) {
	var pager string
	var err error

	pager = os.ExpandEnv("$PAGER")
	if pager != "" {
		return pager, nil
	}

	pager, err = exec.LookPath("less")
	if err == nil {
		return pager, nil
	}

	pager, err = exec.LookPath("more")
	if err != nil {
		return "", err
	}
	return pager, nil
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
