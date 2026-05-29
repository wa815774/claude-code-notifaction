//go:build windows

package notifier

import (
	"fmt"
	"strings"

	"github.com/777genius/claude-notifications/internal/logging"
)

// sendWindowsToast sends a Toast notification on Windows using PowerShell
// with UTF-16LE + Base64 encoded script to avoid command-line encoding issues.
// Uses CDATA sections to safely embed emoji, Chinese characters, and special characters.
//
// This approach was chosen over beeep/go-toast because go-toast's COM API path
// uses CoRegisterClassObject with a global GUID, which fails in multi-process
// scenarios (concurrent Claude Code sessions). Its PowerShell fallback has
// encoding edge cases on non-English Windows systems.
//
// PowerShell's -EncodedCommand expects base64-encoded UTF-16LE, which ensures
// non-ASCII characters survive command-line argument parsing regardless of the
// system's active code page (CP936, CP932, etc.).
func (n *Notifier) sendWindowsToast(title, message, subtitle, sessionID string, timeSensitive bool) error {
	xml := buildWindowsToastXML(title, message, subtitle, sessionID, timeSensitive)
	psScript := buildWindowsToastScript(xml)
	encodedCmd := encodeUTF16LEBase64(psScript)

	cmd := execCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodedCmd)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("PowerShell toast failed: %w, output: %s", err, outputStr)
	}
	// Some PowerShell errors don't set $LASTEXITCODE but write to stderr.
	if outputStr != "" && strings.Contains(strings.ToLower(outputStr), "error") {
		return fmt.Errorf("PowerShell toast produced error output: %s", outputStr)
	}

	logging.Debug("Windows toast sent via PowerShell -EncodedCommand: title=%s", title)
	return nil
}
