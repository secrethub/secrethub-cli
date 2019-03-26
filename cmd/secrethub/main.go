package main

import (
	"fmt"
	"os"

	"github.com/keylockerbv/secrethub-cli/internals/secrethub"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// These variables are set at compile-time using ldflags when creating a build.
var (
	version string
	commit  string
)

func main() {
	err := secrethub.NewApp().Version(version, commit).Run(os.Args[1:])
	if err != nil {
		handleError(err)
	}

	os.Exit(0)
}

// handleError will process the error.
// If the user wants to then a bug report is sent.
func handleError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encountered an error: %s\n", err)

		// We need to block or we will exit before the bug report is sent.
		errio.CaptureErrorAndWait(err, nil)

		os.Exit(1)
	}
}
