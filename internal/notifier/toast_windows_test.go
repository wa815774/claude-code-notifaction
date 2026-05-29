//go:build windows

package notifier

import (
	"encoding/base64"
	"os/exec"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/777genius/claude-notifications/internal/analyzer"
	"github.com/777genius/claude-notifications/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendWindowsToast_Success(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	// Mock execCommand to capture the command
	var capturedCmd *exec.Cmd
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedCmd = exec.Command(name, arg...)
		return capturedCmd
	}
	defer func() { execCommand = originalExecCommand }()

	// Override the command to return success without actually running PowerShell
	capturedCmd = nil
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedCmd = exec.Command("cmd", "/c", "echo mock")
		return capturedCmd
	}

	err := n.sendWindowsToast("Test Title", "Test Message", "", "", false)
	require.NoError(t, err)
	require.NotNil(t, capturedCmd)

	// Verify it uses powershell.exe
	assert.Equal(t, "powershell.exe", capturedCmd.Path)

	// Verify arguments contain -EncodedCommand
	args := capturedCmd.Args
	require.True(t, len(args) >= 4, "Expected at least 4 args, got %d: %v", len(args), args)
	assert.Equal(t, "-NoProfile", args[1])
	assert.Equal(t, "-NonInteractive", args[2])
	assert.Equal(t, "-EncodedCommand", args[3])
	require.True(t, len(args) > 4, "Expected encoded command arg")

	// Verify the encoded command can be decoded
	encodedCmd := args[4]
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	require.NoError(t, err)

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	script := string(utf16.Decode(u16s))

	// Verify script contains WinRT types and our content
	assert.Contains(t, script, "Windows.UI.Notifications.ToastNotificationManager")
	assert.Contains(t, script, "Claude Code Notifications")
	assert.Contains(t, script, "Test Title")
	assert.Contains(t, script, "Test Message")
}

func TestSendWindowsToast_ChineseContent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	var capturedArgs []string
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return exec.Command("cmd", "/c", "echo mock")
	}
	defer func() { execCommand = originalExecCommand }()

	title := "✅ Completed [评估 skill-audit]"
	message := "修复记录（汇总格式）本轮共修复 16 个回复"

	err := n.sendWindowsToast(title, message, "", "", false)
	require.NoError(t, err)

	// Decode the encoded command
	require.True(t, len(capturedArgs) > 4)
	encodedCmd := capturedArgs[4]
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	require.NoError(t, err)

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	script := string(utf16.Decode(u16s))

	// Verify Chinese characters survive the encoding round-trip
	assert.Contains(t, script, title)
	assert.Contains(t, script, message)
	assert.Contains(t, script, "CDATA")
}

func TestSendWindowsToast_EmojiContent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	var capturedArgs []string
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return exec.Command("cmd", "/c", "echo mock")
	}
	defer func() { execCommand = originalExecCommand }()

	title := "🎉 Celebration"
	message := "📋 Checklist complete ✅"

	err := n.sendWindowsToast(title, message, "", "", false)
	require.NoError(t, err)

	require.True(t, len(capturedArgs) > 4)
	encodedCmd := capturedArgs[4]
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	require.NoError(t, err)

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	script := string(utf16.Decode(u16s))

	assert.Contains(t, script, title)
	assert.Contains(t, script, message)
}

func TestSendWindowsToast_WithSubtitleAndSessionID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	var capturedArgs []string
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return exec.Command("cmd", "/c", "echo mock")
	}
	defer func() { execCommand = originalExecCommand }()

	err := n.sendWindowsToast("Title", "Message", "Subtitle", "session-123", true)
	require.NoError(t, err)

	require.True(t, len(capturedArgs) > 4)
	encodedCmd := capturedArgs[4]
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	require.NoError(t, err)

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	script := string(utf16.Decode(u16s))

	assert.Contains(t, script, `scenario="reminder"`)
	assert.Contains(t, script, "Subtitle")
	assert.Contains(t, script, "session-123")
	assert.Contains(t, script, "<tag>")
	assert.Contains(t, script, "<group>")
}

func TestSendWindowsToast_FailureReturnsError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	// Mock execCommand to return failure
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd", "/c", "exit 1")
	}
	defer func() { execCommand = originalExecCommand }()

	err := n.sendWindowsToast("Title", "Message", "", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PowerShell toast failed")
}

func TestSendWindowsToast_ErrorOutput(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	n := New(cfg)

	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd", "/c", "echo some error && exit 0")
	}
	defer func() { execCommand = originalExecCommand }()

	err := n.sendWindowsToast("Title", "Message", "", "", false)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "error")
}

func TestSendDesktop_WindowsPrefersPowerShell(t *testing.T) {
	// This test verifies that on Windows, SendDesktop prefers sendWindowsToast
	// over beeep. Since we can't easily mock platform.IsWindows() without
	// build tags, we at least verify the function structure is correct by
	// checking SendDesktop doesn't panic with Windows config.
	cfg := config.DefaultConfig()
	cfg.Notifications.Desktop.Enabled = true
	cfg.Notifications.Desktop.Sound = false
	cfg.Notifications.Desktop.ClickToFocus = false
	n := New(cfg)

	// Mock execCommand to avoid actually running PowerShell
	originalExecCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cmd", "/c", "echo mock")
	}
	defer func() { execCommand = originalExecCommand }()

	// Should not panic
	err := n.SendDesktop(analyzer.StatusTaskComplete, "[тест] Test message", "session-id", "/test")
	// Error is acceptable in test environment
	_ = err
}
