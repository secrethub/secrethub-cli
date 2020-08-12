package secrethub

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// CredentialListCommand creates a backup code to restore a credential from a code.
type CredentialListCommand struct {
	io            ui.IO
	newClient     newClientFunc
	useTimestamps bool
}

// NewAccountInitCommand creates a new CredentialListCommand.
func NewCredentialListCommand(io ui.IO, newClient newClientFunc) *CredentialListCommand {
	return &CredentialListCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CredentialListCommand) Register(r command.Registerer) {
	clause := r.CreateCommand("ls", "List all your credentials.")
	clause.Alias("list")

	//TODO
	//registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, nil, cmd.Run)
}

// Run lists all the currently authenticated account's credentials.
func (cmd *CredentialListCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	timeFormatter := NewTimeFormatter(cmd.useTimestamps)

	w := tabwriter.NewWriter(cmd.io.Output(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(w,
		"FINGERPRINT\t"+
			"TYPE\t"+
			"ENABLED\t"+
			"CREATED\t"+
			"DESCRIPTION")

	it := client.Credentials().List(&secrethub.CredentialListParams{})
	for {
		cred, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		enabled := "no"
		if cred.Enabled {
			enabled = "yes"
		}

		row := []string{
			cred.Fingerprint[:16],
			string(cred.Type),
			enabled,
			timeFormatter.Format(cred.CreatedAt),
			cred.Description,
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}
