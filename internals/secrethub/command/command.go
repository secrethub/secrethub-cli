package command

import (
	"github.com/alecthomas/kingpin"

	"github.com/secrethub/secrethub-cli/internals/cli"
)

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *cli.CommandClause
}

// BindAction binds a function to a command clause, so that
// it is executed when the command is parsed.
func BindAction(clause *cli.CommandClause, fn func() error) {
	clause.Action(
		func(*kingpin.ParseContext) error {
			return fn()
		},
	)
}
