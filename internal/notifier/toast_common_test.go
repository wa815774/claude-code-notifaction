package notifier

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestBuildWindowsToastXML_Basic(t *testing.T) {
	xml := buildWindowsToastXML("Test Title", "Test Message", "", "", false)

	if !strings.Contains(xml, `<toast>`) {
		t.Error("XML should contain <toast>")
	}
	if !strings.Contains(xml, `</text>`) {
		t.Error("XML should contain text elements")
	}
	if !strings.Contains(xml, `<![CDATA[Test Title]]>`) {
		t.Error("Title should be wrapped in CDATA")
	}
	if !strings.Contains(xml, `<![CDATA[Test Message]]>`) {
		t.Error("Message should be wrapped in CDATA")
	}
}

func TestBuildWindowsToastXML_TimeSensitive(t *testing.T) {
	xml := buildWindowsToastXML("Title", "Msg", "", "", true)
	if !strings.Contains(xml, `scenario="reminder"`) {
		t.Error("XML should contain time-sensitive scenario")
	}
}

func TestBuildWindowsToastXML_WithSubtitle(t *testing.T) {
	xml := buildWindowsToastXML("Title", "Msg", "Subtitle", "", false)
	// Should have 3 text elements: title, subtitle, message
	count := strings.Count(xml, `<text>`)
	if count != 3 {
		t.Errorf("Expected 3 text elements, got %d", count)
	}
	if !strings.Contains(xml, `<![CDATA[Subtitle]]>`) {
		t.Error("Subtitle should be wrapped in CDATA")
	}
}

func TestBuildWindowsToastXML_WithSessionID(t *testing.T) {
	xml := buildWindowsToastXML("Title", "Msg", "", "session-123", false)
	if !strings.Contains(xml, `<tag>session-123</tag>`) {
		t.Error("XML should contain tag element")
	}
	if !strings.Contains(xml, `<group>session-123</group>`) {
		t.Error("XML should contain group element")
	}
}

func TestBuildWindowsToastXML_ChineseCharacters(t *testing.T) {
	title := "✅ Completed [评估 skill-audit]"
	message := "修复记录（汇总格式）本轮共修复 16 个回复"
	xml := buildWindowsToastXML(title, message, "", "", false)

	if !strings.Contains(xml, `<![CDATA[`+title+`]]>`) {
		t.Error("Chinese title should be preserved in CDATA")
	}
	if !strings.Contains(xml, `<![CDATA[`+message+`]]>`) {
		t.Error("Chinese message should be preserved in CDATA")
	}
}

func TestBuildWindowsToastXML_Emoji(t *testing.T) {
	title := "🎉 Celebration"
	message := "📋 Checklist complete ✅"
	xml := buildWindowsToastXML(title, message, "", "", false)

	if !strings.Contains(xml, title) {
		t.Error("Emoji title should be preserved in CDATA")
	}
	if !strings.Contains(xml, message) {
		t.Error("Emoji message should be preserved in CDATA")
	}
}

func TestBuildWindowsToastXML_SpecialXMLChars(t *testing.T) {
	title := `Title <script>alert("xss")</script>`
	message := `Message & more <br/>`
	xml := buildWindowsToastXML(title, message, "", "", false)

	// CDATA should preserve the raw characters
	if !strings.Contains(xml, `<![CDATA[`+title+`]]>`) {
		t.Error("Special XML chars in title should be preserved in CDATA")
	}
	if !strings.Contains(xml, `<![CDATA[`+message+`]]>`) {
		t.Error("Special XML chars in message should be preserved in CDATA")
	}
}

func TestBuildWindowsToastScript_ContainsXML(t *testing.T) {
	xml := `<toast><visual>...</visual></toast>`
	script := buildWindowsToastScript(xml)

	if !strings.Contains(script, xml) {
		t.Error("Script should contain the XML content")
	}
	if !strings.Contains(script, "Windows.UI.Notifications.ToastNotificationManager") {
		t.Error("Script should load WinRT types")
	}
	if !strings.Contains(script, "$APP_ID = 'Claude Code Notifications'") {
		t.Error("Script should set AppID")
	}
	if !strings.Contains(script, "CreateToastNotifier") {
		t.Error("Script should create toast notifier")
	}
}

func TestBuildWindowsToastScript_HereStringFormat(t *testing.T) {
	xml := `<toast><visual></visual></toast>`
	script := buildWindowsToastScript(xml)

	// Verify here-string syntax
	if !strings.Contains(script, `$template = @"`) {
		t.Error("Script should use here-string for XML")
	}
	if !strings.Contains(script, `"@`) {
		t.Error("Script should close here-string")
	}
}

func TestEncodeUTF16LEBase64_ASCII(t *testing.T) {
	input := "Hello World"
	encoded := encodeUTF16LEBase64(input)

	// Decode and verify
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	// Convert UTF-16LE bytes back to string
	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	result := string(utf16.Decode(u16s))

	if result != input {
		t.Errorf("Decoded result = %q, want %q", result, input)
	}
}

func TestEncodeUTF16LEBase64_Chinese(t *testing.T) {
	input := "修复记录（汇总格式）"
	encoded := encodeUTF16LEBase64(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	result := string(utf16.Decode(u16s))

	if result != input {
		t.Errorf("Decoded result = %q, want %q", result, input)
	}
}

func TestEncodeUTF16LEBase64_Emoji(t *testing.T) {
	input := "🎉📋✅"
	encoded := encodeUTF16LEBase64(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	result := string(utf16.Decode(u16s))

	if result != input {
		t.Errorf("Decoded result = %q, want %q", result, input)
	}
}

func TestEncodeUTF16LEBase64_Mixed(t *testing.T) {
	input := "Hello 世界 🌍! 修复完成"
	encoded := encodeUTF16LEBase64(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	u16s := make([]uint16, len(decodedBytes)/2)
	for i := 0; i < len(decodedBytes)/2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}
	result := string(utf16.Decode(u16s))

	if result != input {
		t.Errorf("Decoded result = %q, want %q", result, input)
	}
}

func TestEncodeUTF16LEBase64_LittleEndian(t *testing.T) {
	// Verify the encoding is actually little-endian.
	// For ASCII 'A' (0x0041), UTF-16LE should be [0x41, 0x00].
	input := "A"
	encoded := encodeUTF16LEBase64(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	if len(decodedBytes) != 2 {
		t.Fatalf("Expected 2 bytes for single ASCII char, got %d", len(decodedBytes))
	}

	// Little-endian: low byte first
	if decodedBytes[0] != 0x41 {
		t.Errorf("First byte = 0x%02x, want 0x41 (little-endian)", decodedBytes[0])
	}
	if decodedBytes[1] != 0x00 {
		t.Errorf("Second byte = 0x%02x, want 0x00", decodedBytes[1])
	}
}

func TestEncodeUTF16LEBase64_SurrogatePairs(t *testing.T) {
	// Emoji 🎉 = U+1F389 = surrogate pair D83C DF89 in UTF-16
	input := "🎉"
	encoded := encodeUTF16LEBase64(input)

	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	// Should be 4 bytes (2 UTF-16 code units)
	if len(decodedBytes) != 4 {
		t.Fatalf("Expected 4 bytes for emoji surrogate pair, got %d", len(decodedBytes))
	}

	u16s := make([]uint16, 2)
	for i := 0; i < 2; i++ {
		u16s[i] = uint16(decodedBytes[i*2]) | uint16(decodedBytes[i*2+1])<<8
	}

	// Verify surrogate pair values
	if u16s[0] != 0xD83C {
		t.Errorf("First surrogate = 0x%04x, want 0xD83C", u16s[0])
	}
	if u16s[1] != 0xDF89 {
		t.Errorf("Second surrogate = 0x%04x, want 0xDF89", u16s[1])
	}

	// Verify round-trip
	result := string(utf16.Decode(u16s))
	if result != input {
		t.Errorf("Round-trip result = %q, want %q", result, input)
	}
}
