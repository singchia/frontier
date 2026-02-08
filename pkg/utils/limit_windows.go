//go:build windows
// +build windows

package utils

// SetRLimit is a no-op on Windows as it doesn't support Unix-style rlimit.
// Windows handles file descriptor limits differently through system settings.
func SetRLimit(fileLimit uint64) error {
	// Windows doesn't support Unix-style rlimit, so we just return nil
	// to indicate "success" without doing anything.
	// File descriptor limits on Windows are managed by the system.
	return nil
}
