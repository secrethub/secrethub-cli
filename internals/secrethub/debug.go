package secrethub

import (
	"strconv"

	"github.com/keylockerbv/secrethub-cli/internals/cli"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// RegisterDebugFlag registers a debug flag that changes the log level of the given logger to DEBUG.
func RegisterDebugFlag(r FlagRegisterer, logger cli.Logger) {
	flag := debugFlag{
		logger: logger,
	}
	r.Flag("debug", "Enable debug mode.").Short('D').SetValue(&flag)
}

// debugFlag configures the debug level of a logger.
type debugFlag struct {
	debug  bool
	logger cli.Logger
}

func (f debugFlag) init() {
	if f.debug {
		f.logger.EnableDebug()
		f.logger.Debugf("Loglevel set to debug")
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
		return errio.Error(err)
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
