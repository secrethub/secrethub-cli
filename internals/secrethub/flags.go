package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/spf13/pflag"
)

// FlagRegisterer allows others to register flags on it.
type FlagRegisterer interface {
	Flag(name, help string) *cli.Flag
}

func registerTimestampFlag(r FlagRegisterer) *pflag.Flag {
	return r.Flag("timestamp", "Show timestamps formatted to RFC3339 instead of human readable durations.").Short('T').Flag
}

func registerForceFlag(r FlagRegisterer) *pflag.Flag {
	return r.Flag("force", "Ignore confirmation and fail instead of prompt for missing arguments.").Short('f').Flag
}
