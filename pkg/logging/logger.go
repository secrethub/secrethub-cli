package logging

// Logger can be used to log information.
type Logger interface {
	Debug(format string, args ...interface{})
}
