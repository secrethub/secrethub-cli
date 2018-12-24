package cli

import (
	"os"

	logging "github.com/op/go-logging"
)

// Injected Variables
var (
	// SentryDSN is the dsn used to report errors.
	// This is injected by the Make file.
	// Because we use a production DSN and development DSN.
	ServerSentryDSN string
	ClientSentryDSN string
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
