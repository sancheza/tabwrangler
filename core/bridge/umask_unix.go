//go:build !windows

package bridge

import "syscall"

// restrictUmask sets a umask that removes all group and other permissions and
// returns a function that restores the previous value.
func restrictUmask() func() {
	old := syscall.Umask(0177)
	return func() { syscall.Umask(old) }
}
