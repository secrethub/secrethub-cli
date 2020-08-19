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
func BindAction(clause *cli.CommandClause, params []cli.ArgValue, fn func() error) {
	if params != nil {
		clause.Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			return cli.ArgumentRegister(params, args)
		}
	}
	if fn != nil {
		clause.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}

func BindActionArr(clause *cli.CommandClause, param cli.ArgArrValue, fn func() error) {
	if param != nil {
		clause.Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			return param.Set(args)
		}
	}
	if fn != nil {
		clause.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fn()
		}
	}
}
