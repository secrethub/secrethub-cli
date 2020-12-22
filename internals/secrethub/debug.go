package secrethub

import (
	"strconv"

	"github.com/secrethub/secrethub-cli/internals/cli"
)

// RegisterDebugFlag registers a debug flag that changes the log level of the given logger to DEBUG.
func RegisterDebugFlag(app *cli.App, logger cli.Logger) {
	commandClause := cli.CommandClause{
		Cmd: app.Cmd,
		App: app,
	}
	flag := debugFlag{
		logger: logger,
	}
	commandClause.PersistentFlags().VarP(&flag, "debug", "D", "Enable debug mode.")
	commandClause.Flag("debug").NoOptDefVal = "true"
}

// debugFlag configures the debug level of a logger.
type debugFlag struct {
	debug  bool
	logger cli.Logger
}

func (f debugFlag) Type() string {
	return "debugFlag"
}

func (f debugFlag) init() {
	if f.debug {
		f.logger.EnableDebug()
	}
}

// String implements the flag.Value interface.
func (f debugFlag) String() string {
	return strconv.FormatBool(f.debug)
}

// Set changes the log level to debug when the given value is true.
func (f *debugFlag) Set(value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.debug = b
	f.init()
	return nil
}

// IsBoolFlag makes the flag a boolean flag when used in a Kingpin application.
// Thus, the flag can be used without argument (--debug or -D).
func (f debugFlag) IsBoolFlag() bool {
	return true
}
