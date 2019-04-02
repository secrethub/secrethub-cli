// +build linux darwin

package cloneproc

import (
	"os"
	"os/exec"
	"syscall"
)

// Spawn starts a detached clone of the client with the supplied parameters.
func Spawn(args ...string) error {
	cmd := exec.Command(os.Args[0], args...)

	// Detach spawned process from the current
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Start()
}
