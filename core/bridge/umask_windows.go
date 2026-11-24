//go:build windows

package bridge

// restrictUmask is a no-op on Windows, which has no umask.
func restrictUmask() func() { return func() {} }
