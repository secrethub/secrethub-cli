package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/spf13/cobra"
)

// RegisterDebugFlag registers a debug flag that changes the log level of the given logger to DEBUG.
func RegisterDebugFlag(app *cli.App, logger cli.Logger) {
	app.PersistentFlags().BoolP("debug", "D", false, "Enable debug mode.")
	app.Root.Cmd.PersistentPreRunE = cli.MergeFuncs(app.Root.Cmd.PersistentPreRunE, func(command *cobra.Command, strings []string) error {
		if app.Root.Cmd.Flag("debug").Changed {
			logger.EnableDebug()
		}
		return nil
	})
}
