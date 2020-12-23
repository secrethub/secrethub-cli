package secrethub

import (
	"github.com/fatih/color"
	"github.com/secrethub/secrethub-cli/internals/cli"
)

// RegisterColorFlag registers a color flag that configures whether colored output is used.
func RegisterColorFlag(app *cli.App) {
	app.PersistentFlags().BoolVar(&color.NoColor, "no-color", false, "Disable colored output.")
}
