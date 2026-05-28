package sessiontitle

import (
	"testing"

	"github.com/wa815774/claude-notifications/pkg/jsonl"
)

func TestExtractTitle_CustomTitle(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "custom-title", CustomTitle: "我的自定义标题"},
	}
	got := ExtractTitle(messages)
	want := "我的自定义标题"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_AITitle(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "ai-title", AITitle: "AI 生成标题"},
	}
	got := ExtractTitle(messages)
	want := "AI 生成标题"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_Slug(t *testing.T) {
	messages := []jsonl.Message{
		{Slug: "my-session-slug"},
	}
	got := ExtractTitle(messages)
	want := "my-session-slug"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_FirstUserPrompt(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "user", Message: jsonl.MessageContent{ContentString: "帮我写个通知插件"}},
	}
	got := ExtractTitle(messages)
	want := "帮我写个通知插件"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_Priority(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "ai-title", AITitle: "AI 标题"},
		{Type: "custom-title", CustomTitle: "自定义标题"},
	}
	got := ExtractTitle(messages)
	want := "自定义标题"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_SkipMetaUser(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "user", IsMeta: true, Message: jsonl.MessageContent{ContentString: "meta 消息"}},
		{Type: "user", Message: jsonl.MessageContent{ContentString: "真实用户消息"}},
	}
	got := ExtractTitle(messages)
	want := "真实用户消息"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_SkipSystemMessage(t *testing.T) {
	messages := []jsonl.Message{
		{Type: "user", Message: jsonl.MessageContent{ContentString: "<system>指令</system>"}},
		{Type: "user", Message: jsonl.MessageContent{ContentString: "真实用户消息"}},
	}
	got := ExtractTitle(messages)
	want := "真实用户消息"
	if got != want {
		t.Errorf("ExtractTitle() = %q, want %q", got, want)
	}
}

func TestExtractTitle_Empty(t *testing.T) {
	got := ExtractTitle([]jsonl.Message{})
	if got != "" {
		t.Errorf("ExtractTitle([]) = %q, want empty string", got)
	}
}

func TestTruncateTitle_Chinese(t *testing.T) {
	got := TruncateTitle("使用playwright操作chrome浏览器", 10)
	want := "使用playwrig…"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}

func TestTruncateTitle_English(t *testing.T) {
	got := TruncateTitle("Configure Playwright MCP", 10)
	want := "Configure …"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}

func TestTruncateTitle_Short(t *testing.T) {
	got := TruncateTitle("短标题", 10)
	want := "短标题"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}

func TestTruncateTitle_ExactLength(t *testing.T) {
	got := TruncateTitle("精确长度", 4)
	want := "精确长度"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}
