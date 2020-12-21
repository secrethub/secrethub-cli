package command

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
)

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *cli.CommandClause
}
