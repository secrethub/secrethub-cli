package secrethub

import (
	"github.com/keylockerbv/secrethub-cli/pkg/cli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func registerTimestampFlag(r cli.FlagRegisterer) *kingpin.FlagClause {
	return r.Flag("timestamp", "Show timestamps formatted to RFC3339 instead of human readable durations.").Short('T')
}

func registerForceFlag(r cli.FlagRegisterer) *kingpin.FlagClause {
	return r.Flag("force", "Ignore confirmation and fail instead of prompt for missing arguments.").Short('f')
}
