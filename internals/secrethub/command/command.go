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
func BindAction(clause *cli.CommandClause, argumentRegister func(c *cobra.Command, args []string) error, fn func() error) {
	if argumentRegister != nil {
		clause.PreRunE = argumentRegister
	}
	if fn != nil {
		clause.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}
