package main

import (
	"fmt"
	"os"

	"github.com/secrethub/secrethub-cli/internals/secrethub"
)

func main() {
	err := secrethub.NewApp().Version(secrethub.Version, secrethub.Commit).Run()
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
		os.Exit(1)
	}
}
