package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"

	"github.com/alecthomas/kingpin"
)

// BindAction binds a function to a command clause, so that
// it is executed when the command is parsed.
func BindAction(clause *cli.CommandClause, fn func() error) {
	clause.Action(
		func(*kingpin.ParseContext) error {
			return fn()
		},
	)
}
