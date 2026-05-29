//go:build !windows

package notifier

import "fmt"

// sendWindowsToast is a no-op on non-Windows platforms.
func (n *Notifier) sendWindowsToast(title, message, subtitle, sessionID string, timeSensitive bool) error {
	return fmt.Errorf("sendWindowsToast is Windows-only")
}
