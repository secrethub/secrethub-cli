package spawnprocess

import (
	"os"
	"os/exec"
)

// SpawnCloneProcess starts a detached clone of the client with the supplied parameters.
func SpawnCloneProcess(args ...string) error {
	cmd := exec.Command(os.Args[0], args...)

	return cmd.Start()
}
