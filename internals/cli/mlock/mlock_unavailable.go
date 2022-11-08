//go:build android || darwin || nacl || netbsd || plan9 || windows

package mlock

func init() {
	available = false
}

// As there is no good way to do this on the unsupported systems, we will simply return nil here.
// We do not want the code execution to fail, because we run it on a less compatible system.
func lockMemory() error {
	return nil
}
