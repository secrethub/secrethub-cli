package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/spf13/cobra"
)

// RegisterDebugFlag registers a debug flag that changes the log level of the given logger to DEBUG.
func RegisterDebugFlag(app *cli.App, logger cli.Logger) {
	var flag bool
	app.PersistentFlags().BoolVarP(&flag, "debug", "D", false, "Enable debug mode.")
	app.Root.AddPersistentPreRunE(func(command *cobra.Command, strings []string) error {
		if flag {
			logger.EnableDebug()
		}
		return nil
	})
}
