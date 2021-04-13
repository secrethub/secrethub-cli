package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/mlock"
	"github.com/spf13/cobra"
)

// RegisterMlockFlag registers a mlock flag that enables memory locking when set to true.
func RegisterMlockFlag(app *cli.App) {
	var flag bool
	app.PersistentFlags().BoolVar(&flag, "mlock", false, "Enable memory locking.").Hidden()
	app.Root.AddPersistentPreRunE(func(command *cobra.Command, strings []string) error {
		if flag {
			if mlock.Supported() {
				err := mlock.LockMemory()
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
