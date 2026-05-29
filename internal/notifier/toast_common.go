package notifier

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf16"
)

// buildWindowsToastXML constructs the Windows Toast notification XML.
// Uses CDATA sections to safely embed emoji, Chinese characters, and special
// characters that would otherwise break XmlDocument.LoadXml.
func buildWindowsToastXML(title, message, subtitle, sessionID string, timeSensitive bool) string {
	var xmlContent strings.Builder
	xmlContent.WriteString(`<toast`)
	if timeSensitive {
		xmlContent.WriteString(` scenario="reminder"`)
	}
	xmlContent.WriteString(`><visual><binding template="ToastGeneric">`)

	// Title (first line, large font)
	xmlContent.WriteString(`<text><![CDATA[` + title + `]]></text>`)

	// Subtitle (second line) if present
	if subtitle != "" {
		xmlContent.WriteString(`<text><![CDATA[` + subtitle + `]]></text>`)
	}

	// Message body (last line)
	if message != "" {
		xmlContent.WriteString(`<text><![CDATA[` + message + `]]></text>`)
	}

	xmlContent.WriteString(`</binding></visual>`)
	if sessionID != "" {
		xmlContent.WriteString(`<tag>` + sessionID + `</tag><group>` + sessionID + `</group>`)
	}
	xmlContent.WriteString(`</toast>`)

	return xmlContent.String()
}

// buildWindowsToastScript creates the PowerShell script that displays the toast.
// The script uses a here-string ($template = @"..."@) for the XML so quotes
// inside the XML do not need escaping.
func buildWindowsToastScript(xml string) string {
	return fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] | Out-Null
$APP_ID = 'Claude Code Notifications'
$template = @"
%s
"@
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($APP_ID).Show($toast)
`, xml)
}

// encodeUTF16LEBase64 encodes a string as base64(UTF-16LE).
// This is the format expected by PowerShell's -EncodedCommand parameter.
func encodeUTF16LEBase64(s string) string {
	u16s := utf16.Encode([]rune(s))
	utf16Bytes := make([]byte, len(u16s)*2)
	for i, u := range u16s {
		utf16Bytes[i*2] = byte(u)
		utf16Bytes[i*2+1] = byte(u >> 8)
	}
	return base64.StdEncoding.EncodeToString(utf16Bytes)
}
