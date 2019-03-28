package cli

import (
	"os"

	logging "github.com/op/go-logging"
)

type Logger interface {
	// Debugf logs a message when debug mode is enabled.
	Debugf(format string, args ...interface{})
	// EnableDebug turns printing debug messages on.
	EnableDebug()
}

type logger struct {
	*logging.Logger
}

// NewLogger returns a logger with the given format, module and loglevel.
func NewLogger(format string, module string, debug bool) Logger {
	formatter := logging.MustStringFormatter(format)
	backend := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), formatter)
	logging.SetBackend(backend)

	l := logging.MustGetLogger(module)
	if debug {
		logging.SetLevel(logging.DEBUG, module)
		l.Debug("Loglevel set to debug")
	} else {
		logging.SetLevel(logging.INFO, module)
	}

	return logger{Logger: l}
}

// EnableDebug turns printing debug messages on.
func (l logger) EnableDebug() {
	logging.SetLevel(logging.DEBUG, l.Module)
}
