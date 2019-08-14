// +build dragonfly freebsd linux openbsd solaris

package mlock

import (
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func init() {
	available = true
}

func lockMemory() error {
	err := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)

	if err == unix.ENOSYS {
		return errMlock.Code("enosys").Errorf(
			"%s\n\n"+
				"Could not prevent memory from being written to swap because mlock() is not implemented on this system.\n"+
				"This usually means that the mlock syscall is not available.\n"+
				"This requires root privileges as well as a system that supports mlock.\n"+
				"Please enable mlock on your system, try again with root privileges,",
			err)
	} else if err == unix.ENOMEM {
		execFile, _ := os.Executable()
		execAbs, _ := filepath.Abs(execFile)

		return errMlock.Code("enomem").Errorf(
			"%s\n\n"+
				"Could not prevent memory from being written to swap because mlock() failed with ENOMEM.\n"+
				"This probably means that the user is not allowed to lock enough memory.\n"+
				"Either execute as root, or give your user the right privileges by editing the /etc/security/limits.conf file.\n"+
				"On Linux systems, the secrets executable can be given the correct privileges by executing `sudo setcap 'cap_ipc_lock=+ep' %s`.\n"+
				"It can also mean that the system is out of memory. If this is the case,\n"+
				"try freeing up some RAM by closing down some applications and see if the problem persists.\n",
			err, execAbs)
	} else if err == unix.EPERM {
		return errMlock.Code("eperm").Errorf(
			"%s\n\n"+
				"Could not prevent memory from being written to swap because your are not privileged for this operation.\n"+
				"This usually means that you need root privileges.\n"+
				"Please try again with root privileges and see if the problem persists.",
			err)
	} else if err != nil {
		return err
	}
	log.Debugf("mlock is active")
	return nil
}
