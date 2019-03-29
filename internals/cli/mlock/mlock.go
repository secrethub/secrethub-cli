// Package mlock allows for locking memory, providing implementations for different operating systems.
package mlock

import (
	"github.com/keylockerbv/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-go/internals/errio"
)

var (
	log = cli.NewLogger()

	// This should be set per OS
	available bool

	errMlock = errio.Namespace("mlock")

	// ErrNotSupported is returned when mlock is not available for the platform
	ErrNotSupported = errMlock.Code("not_supported").Error("mlock is not supported")
)

// Supported returns true if LockMemory is available on the system.
func Supported() bool {
	return available
}

// LockMemory prevents any memory being written to disk as swap.
func LockMemory() error {
	if Supported() {
		return lockMemory()
	}
	return ErrNotSupported
}
