package cloneproc

import (
	"os"
	"os/exec"
)

// Spawn starts a detached clone of the client with the supplied parameters.
func Spawn(args ...string) error {
	cmd := exec.Command(os.Args[0], args...)

	return cmd.Start()
}
