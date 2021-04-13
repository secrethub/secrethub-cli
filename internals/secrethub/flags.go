package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
)

func registerTimestampFlag(r *cli.CommandClause, p *bool) {
	r.Flags().BoolVarP(p, "timestamp", "T", false, "Show timestamps formatted to RFC3339 instead of human readable durations.")
}

func registerForceFlag(r *cli.CommandClause, p *bool) {
	r.Flags().BoolVarP(p, "force", "f", false, "Ignore confirmation and fail instead of prompt for missing arguments.")
}
