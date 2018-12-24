// +build linux darwin

package spawnprocess

import (
	"os"
	"os/exec"
	"syscall"
)

// SpawnCloneProcess starts a detached clone of the client with the supplied parameters.
func SpawnCloneProcess(args ...string) error {
	cmd := exec.Command(os.Args[0], args...)

	// Detach spawned process from the current
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Start()
}
