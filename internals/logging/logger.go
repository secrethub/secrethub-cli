package logging

// Logger can be used to log information.
type Logger interface {
	Debugf(format string, args ...interface{})
}
