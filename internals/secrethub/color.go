package secrethub

import (
	"github.com/fatih/color"
	"github.com/secrethub/secrethub-cli/internals/cli"
)

// RegisterColorFlag registers a color flag that configures whether colored output is used.
func RegisterColorFlag(app *cli.App) {
	commandClause := cli.CommandClause{
		Cmd: app.Cmd,
		App: app,
	}
	commandClause.PersistentFlags().BoolVar(&color.NoColor, "no-color", false, "Disable colored output.")
}
