package cli

import (
	"os"

	logging "github.com/op/go-logging"
)

const logFormat = `%{color}%{level:.4s} ▶ %{color:reset} %{message}`

type Logger interface {
	// Debugf logs a message when debug mode is enabled.
	Debugf(format string, args ...interface{})
	// EnableDebug turns printing debug messages on.
	EnableDebug()
}

type logger struct {
	*logging.Logger
}

func init() {
	formatter := logging.MustStringFormatter(logFormat)
	backend := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), formatter)
	logging.SetBackend(backend)
}

// NewLogger returns a logger with the given format, module and loglevel.
func NewLogger(module string) Logger {
	l := logging.MustGetLogger(module)
	logging.SetLevel(logging.INFO, module)

	return logger{Logger: l}
}

// EnableDebug turns printing debug messages on.
func (l logger) EnableDebug() {
	logging.SetLevel(logging.DEBUG, l.Module)
	l.Debug("Loglevel set to debug")
}
