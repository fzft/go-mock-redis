//go:build linux
// +build linux

package node

import (
	"golang.org/x/sys/unix"
	"strings"
)

func isFDValid(fd int) bool {
	// Try to get the flags of the file descriptor
	_, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	if err != nil {
		return false
	} else {
		return true
	}
}

// IsTemporaryError checks if the error is temporary, e.g., EAGAIN or EWOULDBLOCK.
func IsTemporaryError(err error) bool {
	// This may need more sophisticated checking depending on error format.
	return strings.Contains(err.Error(), "EAGAIN") || strings.Contains(err.Error(), "EWOULDBLOCK")
}

func CloseFd(fd int) error {
	if isFDValid(fd) {
		if err := unix.Close(fd); err != nil {
			return err
		}
	}
	return nil
}
