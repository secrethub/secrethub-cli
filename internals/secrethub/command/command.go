package command

import (
	"github.com/spf13/cobra"

	"github.com/secrethub/secrethub-cli/internals/cli"
)

// Registerer allows others to register commands on it.
type Registerer interface {
	CreateCommand(cmd string, help string) *cli.CommandClause
}

// BindAction binds a function to a command clause, so that
// it is executed when the command is parsed.
func BindAction(clause *cli.CommandClause, prefn func(c *cobra.Command, args []string) error, fn func() error) {
	if prefn != nil {
		clause.Command.RunE = prefn
	}
	clause.Command.PostRunE = func(cmd *cobra.Command, args []string) error {
		return fn()
	}
}
