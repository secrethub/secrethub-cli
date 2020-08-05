package command

import (
	"github.com/spf13/cobra"

	"github.com/secrethub/secrethub-cli/internals/cli"
)

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *cli.CommandClause
}

// BindAction binds a function to a command clause, so that
// it is executed when the command is parsed.
func BindAction(clause *cli.CommandClause, fn func(cmd *cobra.Command, args []string) error) {
	clause.RunE = fn
}
