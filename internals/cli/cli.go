package cli

// Injected Variables
var (
	// SentryDSN is the dsn used to report errors.
	// This is injected by the Make file.
	// Because we use a production DSN and development DSN.
	ServerSentryDSN string
	ClientSentryDSN string
)
