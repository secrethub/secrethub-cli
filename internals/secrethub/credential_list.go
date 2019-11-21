package secrethub

import (
	"fmt"
	"strings"
	"text/tabwriter"

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
	clause := r.Command("ls", "List all your credentials.")
	clause.Alias("list")

	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	command.BindAction(clause, cmd.Run)
}

// Run lists all the currently authenticated account's credentials.
func (cmd *CredentialListCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	credentials, err := client.Credentials().ListMine()
	if err != nil {
		return err
	}

	timeFormatter := NewTimeFormatter(cmd.useTimestamps)

	w := tabwriter.NewWriter(cmd.io.Stdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(w,
		"NAME\t"+
			"TYPE\t"+
			"FINGERPRINT\t"+
			"STATUS\t"+
			"CREATED")

	for _, cred := range credentials {
		row := []string{
			cred.Name,
			string(cred.Type),
			cred.Fingerprint[:16],
			"active",
			timeFormatter.Format(cred.CreatedAt),
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}
