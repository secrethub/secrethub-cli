package cli

import (
	"os"

	logging "github.com/op/go-logging"
)

// NewLogger returns a logger with the given format, module and loglevel.
func NewLogger(format string, module string, debug bool) *logging.Logger {
	formatter := logging.MustStringFormatter(format)
	backend := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), formatter)
	logging.SetBackend(backend)

	logger := logging.MustGetLogger(module)
	if debug {
		logging.SetLevel(logging.DEBUG, module)
		logger.Debug("Loglevel set to debug")
	} else {
		logging.SetLevel(logging.INFO, module)
	}

	return logger
}
