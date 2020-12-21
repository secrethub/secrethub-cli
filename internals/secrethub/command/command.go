package command

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/spf13/cobra"
)

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *cli.CommandClause
}

func BindAction(c *cli.CommandClause, fn func() error) {
	if fn != nil {
		c.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}
