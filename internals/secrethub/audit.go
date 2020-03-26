package secrethub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/pkg/secrethub"

	"github.com/secrethub/secrethub-go/internals/api"
)

var (
	errPagerNotFound = errors.New("no terminal pager available")
)

const (
	pagerEnvvar          = "$PAGER"
	defaultTerminalWidth = 80
)

// AuditCommand is a command to audit a repo or a secret.
type AuditCommand struct {
	io            ui.IO
	path          api.Path
	useTimestamps bool
	timeFormatter TimeFormatter
	newClient     newClientFunc
	perPage       int
	json          bool
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
	clause.Flag("per-page", "number of audit events shown per page").Default("20").Hidden().IntVar(&cmd.perPage)
	clause.Flag("json", "output the audit log in json format").BoolVar(&cmd.json)
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

	var formatter rowFormatter
	if cmd.json {
		formatter = newJSONFormatter(auditTable.header())
	} else {
		terminalWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			terminalWidth = defaultTerminalWidth
		}
		formatter = newColumnFormatter(terminalWidth)
	}

	paginatedWriter, err := newPaginatedWriter(os.Stdout)
	if err != nil {
		return err
	}
	defer paginatedWriter.Close()

	if formatter.printHeader() {
		header, err := formatter.formatRow(auditTable.header())
		if err != nil {
			return err
		}
		fmt.Fprint(paginatedWriter, header)
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

		formattedRow, err := formatter.formatRow(row)
		if err != nil {
			return err
		}

		fmt.Fprint(paginatedWriter, formattedRow)
		if paginatedWriter.IsClosed() {
			break
		}
	}
	return nil
}

type rowFormatter interface {
	printHeader() bool
	formatRow(row []string) (string, error)
}

func newJSONFormatter(fieldNames []string) *jsonFormatter {
	return &jsonFormatter{fields: fieldNames}
}

type jsonFormatter struct {
	fields []string
}

func (f *jsonFormatter) printHeader() bool {
	return false
}

// formatRow returns the json representation of the given row
// with the configured field names as keys and the provided values
func (f *jsonFormatter) formatRow(row []string) (string, error) {
	if len(f.fields) != len(row) {
		return "", fmt.Errorf("unexpected number of json fields")
	}

	jsonMap := make(map[string]string)
	for i, element := range row {
		jsonMap[f.fields[i]] = element
	}

	jsonData, err := json.Marshal(jsonMap)
	if err != nil {
		return "", err
	}
	return string(jsonData) + "\n", nil
}

func newColumnFormatter(tableWidth int) *columnFormatter {
	return &columnFormatter{tableWidth: tableWidth}
}

type columnFormatter struct {
	tableWidth int
}

func (f *columnFormatter) printHeader() bool {
	return true
}

// formatRow formats the given table row to fit the configured width by
// giving each cell an equal width and wrapping the text in cells that exceed it
func (f *columnFormatter) formatRow(row []string) (string, error) {
	maxLinesPerCell := 1
	colWidth := (f.tableWidth - 2*len(row)) / len(row)
	for _, cell := range row {
		lines := len(cell) / colWidth
		if len(cell)%colWidth != 0 {
			lines++
		}
		if lines > maxLinesPerCell {
			maxLinesPerCell = lines
		}
	}

	splitCells := make([][]string, maxLinesPerCell)
	for i := 0; i < maxLinesPerCell; i++ {
		splitCells[i] = make([]string, len(row))
	}

	for i, cell := range row {
		j := 0
		for ; len(cell) > colWidth; j++ {
			splitCells[j][i] = cell[:colWidth]
			cell = cell[colWidth:]
		}
		splitCells[j][i] = cell + strings.Repeat(" ", colWidth-len(cell))
		j++
		for ; j < maxLinesPerCell; j++ {
			splitCells[j][i] = strings.Repeat(" ", colWidth)
		}
	}

	strRes := strings.Builder{}
	for j := 0; j < maxLinesPerCell; j++ {
		strRes.WriteString(strings.Join(splitCells[j], "  ") + "\n")
	}
	return strRes.String(), nil
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

type paginatedWriter struct {
	writer io.WriteCloser
	cmd    *exec.Cmd
	done   <-chan struct{}
	closed bool
}

func (p *paginatedWriter) Write(data []byte) (n int, err error) {
	return p.writer.Write(data)
}

// Close closes the writer to the terminal pager and waits for the terminal pager to close.
func (p *paginatedWriter) Close() error {
	err := p.writer.Close()
	if err != nil {
		return err
	}
	if !p.closed {
		<-p.done
	}
	return nil
}

// IsClosed checks if the terminal pager process has been stopped.
func (p *paginatedWriter) IsClosed() bool {
	if p.closed {
		return true
	}
	select {
	case <-p.done:
		p.closed = true
		return true
	default:
		return false
	}
}

// newPaginatedWriter runs the terminal pager configured in the OS environment
// and returns a writer to its standard input.
func newPaginatedWriter(outputWriter io.Writer) (*paginatedWriter, error) {
	pager, err := pagerCommand()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(pager)

	writer, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stdout = outputWriter
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	done := make(chan struct{}, 1)
	go func() {
		_ = cmd.Wait()
		done <- struct{}{}
	}()
	return &paginatedWriter{writer: writer, cmd: cmd, done: done}, nil
}

// pagerCommand returns the name of the terminal pager configured in the OS environment ($PAGER).
// If no pager is configured less or more is returned depending on which is available.
func pagerCommand() (string, error) {
	var pager string
	var err error

	pager, err = exec.LookPath(os.ExpandEnv(pagerEnvvar))
	if err == nil {
		return pager, nil
	}

	pager, err = exec.LookPath("less")
	if err == nil {
		return pager, nil
	}

	pager, err = exec.LookPath("more")
	if err == nil {
		return pager, nil
	}

	return "", errPagerNotFound
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
