package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/cli"
)

func registerTimestampFlag(r *cli.CommandClause, p *bool) {
	r.BoolVarP(p, "timestamp", "T", false, "Show timestamps formatted to RFC3339 instead of human readable durations.", true, false)
}

func registerForceFlag(r *cli.CommandClause, p *bool) {
	r.BoolVarP(p, "force", "f", false, "Ignore confirmation and fail instead of prompt for missing arguments.", true, false)
}
