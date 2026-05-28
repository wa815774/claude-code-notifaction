package sessiontitle

import (
	"strings"

	"github.com/wa815774/claude-notifications/pkg/jsonl"
)

// ExtractTitle 从消息列表中提取会话标题。
// 优先级：customTitle > aiTitle > slug > firstUserPrompt > ""
func ExtractTitle(messages []jsonl.Message) string {
	var aiTitle, slug, firstUserPrompt string

	for _, msg := range messages {
		// 最高优先级：用户自定义标题
		if msg.Type == "custom-title" && msg.CustomTitle != "" {
			return strings.TrimSpace(msg.CustomTitle)
		}

		// 收集 aiTitle（遇到则记录，继续扫描看是否有 customTitle）
		if msg.Type == "ai-title" && msg.AITitle != "" && aiTitle == "" {
			aiTitle = strings.TrimSpace(msg.AITitle)
		}

		// 收集 slug
		if msg.Slug != "" && slug == "" {
			slug = strings.TrimSpace(msg.Slug)
		}

		// 收集第一条非 meta 用户消息
		if firstUserPrompt == "" && msg.Type == "user" && !msg.IsMeta && msg.Message.ContentString != "" {
			trimmed := strings.TrimSpace(msg.Message.ContentString)
			if trimmed != "" && !strings.HasPrefix(trimmed, "<") {
				firstUserPrompt = trimmed
			}
		}
	}

	// 按优先级返回
	if aiTitle != "" {
		return aiTitle
	}
	if slug != "" {
		return slug
	}
	return firstUserPrompt
}

// TruncateTitle 将标题截断到指定长度。
// 使用 rune 级别的截断以正确处理 Unicode（包括中文）。
// 如果标题长度超过 maxLen，截断后追加 "…"。
func TruncateTitle(title string, maxLen int) string {
	if maxLen <= 0 {
		return title
	}

	runes := []rune(title)
	if len(runes) <= maxLen {
		return title
	}

	return string(runes[:maxLen]) + "…"
}
