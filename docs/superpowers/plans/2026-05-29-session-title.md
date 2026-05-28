# 自定义会话标题实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 从 Claude Code transcript 中提取语义化会话标题（customTitle / aiTitle / slug / firstUserPrompt），替代通知中的随机 session label。

**Architecture:** 新增 `internal/sessiontitle` 包负责标题提取和 Unicode 安全截断；扩展 `jsonl.Message` 结构以支持 transcript 中的标题字段；在 `hooks.sendNotifications` 中集成标题提取逻辑，失败时回退到现有生成标签。

**Tech Stack:** Go 1.22+, 现有测试框架 (go test), beeep (通知库已集成)

---

## 文件变更概览

| 文件 | 操作 | 职责 |
|------|------|------|
| `pkg/jsonl/jsonl.go` | 修改 | `Message` 结构体添加 `CustomTitle`, `AITitle`, `Slug`, `IsMeta` 字段 |
| `internal/sessiontitle/sessiontitle.go` | 创建 | 标题提取逻辑（`ExtractTitle`, `TruncateTitle`） |
| `internal/sessiontitle/sessiontitle_test.go` | 创建 | 标题提取和截断的单元测试 |
| `internal/config/config.go` | 修改 | 添加 `SessionTitleConfig` 和 getter 方法 |
| `internal/config/config_test.go` | 修改 | 新增配置测试 |
| `internal/hooks/hooks.go` | 修改 | `sendNotifications` 签名增加 `messages` 参数，集成标题提取 |
| `internal/hooks/hooks_test.go` | 修改 | 新增集成测试验证标题传递 |

---

### Task 1: 扩展 jsonl.Message 结构体

**Files:**
- Modify: `pkg/jsonl/jsonl.go:13-20`

- [ ] **Step 1: 在 `Message` 结构体中添加标题相关字段**

```go
// Message represents a Claude Code transcript message
type Message struct {
	ParentUUID        string         `json:"parentUuid"`
	Type              string         `json:"type"`
	Message           MessageContent `json:"message"`
	Timestamp         string         `json:"timestamp"`
	IsApiErrorMessage bool           `json:"isApiErrorMessage,omitempty"`
	Error             string         `json:"error,omitempty"`
	// Title-related fields extracted from transcript entries
	CustomTitle string `json:"customTitle,omitempty"`
	AITitle     string `json:"aiTitle,omitempty"`
	Slug        string `json:"slug,omitempty"`
	IsMeta      bool   `json:"isMeta,omitempty"`
}
```

- [ ] **Step 2: 运行 jsonl 包测试确保无回归**

Run: `go test ./pkg/jsonl/... -v`
Expected: PASS (所有现有测试通过)

- [ ] **Step 3: Commit**

```bash
git add pkg/jsonl/jsonl.go
git commit -m "feat(jsonl): 添加 transcript 标题相关字段到 Message 结构体"
```

---

### Task 2: 创建 sessiontitle 包（TDD）

**Files:**
- Create: `internal/sessiontitle/sessiontitle.go`
- Create: `internal/sessiontitle/sessiontitle_test.go`

- [ ] **Step 1: 编写失败测试**

创建 `internal/sessiontitle/sessiontitle_test.go`：

```go
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
	// customTitle 优先级高于 aiTitle
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
	// 以 < 开头的用户消息（通常是系统注入）应被跳过
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
	want := "使用playwright操…"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}

func TestTruncateTitle_English(t *testing.T) {
	got := TruncateTitle("Configure Playwright MCP", 10)
	want := "Configure P…"
	if got != want {
		t.Errorf("TruncateTitle() = %q, want %q", got, want)
	}
}

func TestTruncateTitle_Short(t *testing.T) {
	// 短于 maxLen 的标题不应被截断
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/sessiontitle/... -v`
Expected: FAIL - `undefined: ExtractTitle`, `undefined: TruncateTitle`

- [ ] **Step 3: 实现最小代码使测试通过**

创建 `internal/sessiontitle/sessiontitle.go`：

```go
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/sessiontitle/... -v`
Expected: PASS (所有 12 个测试通过)

- [ ] **Step 5: Commit**

```bash
git add internal/sessiontitle/
git commit -m "feat(sessiontitle): 实现会话标题提取和 Unicode 安全截断"
```

---

### Task 3: 配置层添加 SessionTitleConfig

**Files:**
- Modify: `internal/config/config.go:39-49`
- Modify: `internal/config/config.go:155-165`
- Modify: `internal/config/config.go:376-416`

- [ ] **Step 1: 在 DesktopConfig 中添加 SessionTitle 字段**

在 `internal/config/config.go` 的 `DesktopConfig` 结构体中（第 39-49 行之间），在 `TerminalBundleID` 字段后添加：

```go
// DesktopConfig represents desktop notification settings
type DesktopConfig struct {
	Enabled          bool                `json:"enabled"`
	Sound            bool                `json:"sound"`
	TerminalBell     *bool               `json:"terminalBell"`
	Volume           float64             `json:"volume"`
	AudioDevice      string              `json:"audioDevice"`
	AppIcon          string              `json:"appIcon"`
	ClickToFocus     bool                `json:"clickToFocus"`
	TerminalBundleID string              `json:"terminalBundleId"`
	SessionTitle     *SessionTitleConfig `json:"sessionTitle,omitempty"`
}
```

然后在 `DesktopConfig` 之前（第 38-39 行之间）添加 `SessionTitleConfig` 定义：

```go
// SessionTitleConfig represents session title customization settings
type SessionTitleConfig struct {
	Enabled   *bool `json:"enabled,omitempty"`   // Default: true
	MaxLength *int  `json:"maxLength,omitempty"` // Default: 25
}
```

- [ ] **Step 2: 在 DefaultConfig 中设置默认值**

在 `internal/config/config.go` 的 `DefaultConfig()` 函数中（第 155-165 行，DesktopConfig 初始化部分），在 `TerminalBundleID` 行后添加：

```go
return &Config{
    Notifications: NotificationsConfig{
        Desktop: DesktopConfig{
            Enabled:      true,
            Sound:        true,
            Volume:       1.0,
            TerminalBell: boolPtr(false),
            AppIcon:      filepath.Join(pluginRoot, "claude_icon.png"),
            ClickToFocus: true,
            SessionTitle: &SessionTitleConfig{
                Enabled:   boolPtr(true),
                MaxLength: intPtr(25),
            },
        },
```

- [ ] **Step 3: 在 ApplyDefaults 中补充 sessionTitle 默认值**

在 `internal/config/config.go` 的 `ApplyDefaults()` 方法中（第 376-416 行），在 Desktop defaults 段落后添加：

```go
// SessionTitle defaults
if c.Notifications.Desktop.SessionTitle == nil {
	c.Notifications.Desktop.SessionTitle = &SessionTitleConfig{
		Enabled:   boolPtr(true),
		MaxLength: intPtr(25),
	}
} else {
	if c.Notifications.Desktop.SessionTitle.Enabled == nil {
		c.Notifications.Desktop.SessionTitle.Enabled = boolPtr(true)
	}
	if c.Notifications.Desktop.SessionTitle.MaxLength == nil {
		c.Notifications.Desktop.SessionTitle.MaxLength = intPtr(25)
	}
}
```

- [ ] **Step 4: 添加 getter 方法**

在 `internal/config/config.go` 的末尾（第 650 行之后）添加：

```go
// GetSessionTitleEnabled returns true if custom session title extraction is enabled (default: true)
func (c *Config) GetSessionTitleEnabled() bool {
	if c.Notifications.Desktop.SessionTitle == nil || c.Notifications.Desktop.SessionTitle.Enabled == nil {
		return true // Default: enabled
	}
	return *c.Notifications.Desktop.SessionTitle.Enabled
}

// GetSessionTitleMaxLength returns the maximum length for session titles (default: 25)
func (c *Config) GetSessionTitleMaxLength() int {
	if c.Notifications.Desktop.SessionTitle == nil || c.Notifications.Desktop.SessionTitle.MaxLength == nil {
		return 25 // Default: 25 characters
	}
	return *c.Notifications.Desktop.SessionTitle.MaxLength
}
```

- [ ] **Step 5: 运行 config 包测试确保无回归**

Run: `go test ./internal/config/... -v`
Expected: PASS (所有现有测试通过)

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): 添加 SessionTitleConfig 配置项和 getter 方法"
```

---

### Task 4: Hooks 层集成标题提取

**Files:**
- Modify: `internal/hooks/hooks.go:458-459`
- Modify: `internal/hooks/hooks.go:552`
- Modify: `internal/hooks/hooks.go:768`

- [ ] **Step 1: 修改 sendNotifications 签名**

在 `internal/hooks/hooks.go` 第 768 行，将函数签名从：

```go
func (h *Handler) sendNotifications(status analyzer.Status, body, actions, sessionID, cwd string) {
```

改为：

```go
func (h *Handler) sendNotifications(status analyzer.Status, body, actions, sessionID, cwd string, messages []jsonl.Message) {
```

- [ ] **Step 2: 在 sendNotifications 中集成标题提取**

在 `internal/hooks/hooks.go` 第 772 行（`sessionName := sessionname.GenerateSessionLabel(sessionID)`）之后，添加标题提取逻辑：

```go
sessionName := sessionname.GenerateSessionLabel(sessionID)

// 尝试从 transcript 中提取语义化会话标题
if h.cfg.GetSessionTitleEnabled() && len(messages) > 0 {
	if extracted := sessiontitle.ExtractTitle(messages); extracted != "" {
		sessionName = sessiontitle.TruncateTitle(extracted, h.cfg.GetSessionTitleMaxLength())
		logging.Debug("Using extracted session title: %s", sessionName)
	}
}
```

注意：需要在文件顶部的 import 中添加 `"github.com/wa815774/claude-notifications/internal/sessiontitle"`。

- [ ] **Step 3: 修改所有 sendNotifications 调用点**

**调用点 1**（第 458-459 行，`HandleHook` 中的主调用）：

将：
```go
h.sendNotifications(status, body, actions, hookData.SessionID, hookData.CWD)
```

改为：
```go
h.sendNotifications(status, body, actions, hookData.SessionID, hookData.CWD, parsedMessages)
```

注意：`parsedMessages` 在 `HandleHook` 的 `Stop` 和 `SubagentStop` 分支中已定义。对于 `PreToolUse` 和 `Notification` 分支，`parsedMessages` 为 `nil`（这些 hook 不解析 transcript），所以调用应为：

```go
h.sendNotifications(status, body, actions, hookData.SessionID, hookData.CWD, nil)
```

**调用点 2**（第 552 行，`handleTeammateIdle` 中）：

将：
```go
h.sendNotifications(status, body, "", hookData.SessionID, hookData.CWD)
```

改为：
```go
h.sendNotifications(status, body, "", hookData.SessionID, hookData.CWD, nil)
```

- [ ] **Step 4: 编译检查**

Run: `go build ./...`
Expected: 编译成功，无错误

- [ ] **Step 5: 运行 hooks 包测试确保无回归**

Run: `go test ./internal/hooks/... -v`
Expected: PASS (所有现有测试通过)

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/hooks.go
git commit -m "feat(hooks): 集成语义化会话标题提取到通知流程"
```

---

### Task 5: 添加集成测试

**Files:**
- Modify: `internal/hooks/hooks_test.go`

- [ ] **Step 1: 在 hooks_test.go 中添加标题提取集成测试**

找到 `hooks_test.go` 中现有的 Stop hook 测试，在其后添加新测试：

```go
func TestHandleHook_Stop_WithCustomTitle(t *testing.T) {
	// Create a mock transcript file with customTitle entry
	transcriptDir := t.TempDir()
	transcriptPath := filepath.Join(transcriptDir, "test.jsonl")

	transcriptContent := `{"type":"custom-title","customTitle":"修改任务完成的系统通知"}` + "\n"
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	// Create handler with test config
	pluginRoot := setupTestPlugin(t)
	handler, err := NewHandler(pluginRoot)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}

	// Mock the notifier to capture what gets sent
	mockNotifier := &mockNotifier{}
	handler.notifierSvc = mockNotifier

	// Build hook data
	input := fmt.Sprintf(`{"transcript_path":%q,"session_id":"test-session","cwd":"%s"}`, transcriptPath, transcriptDir)

	// Execute
	if err := handler.HandleHook("Stop", strings.NewReader(input)); err != nil {
		t.Fatalf("HandleHook failed: %v", err)
	}

	// Verify notification was sent with extracted title
	if !mockNotifier.desktopCalled {
		t.Fatal("expected desktop notification to be called")
	}

	// The message should contain the extracted session title
	if !strings.Contains(mockNotifier.lastMessage, "修改任务完成的系统通知") {
		t.Errorf("expected message to contain extracted title, got: %s", mockNotifier.lastMessage)
	}
}

func TestHandleHook_Stop_SessionTitleDisabled(t *testing.T) {
	// Create a mock transcript file with customTitle entry
	transcriptDir := t.TempDir()
	transcriptPath := filepath.Join(transcriptDir, "test.jsonl")

	transcriptContent := `{"type":"custom-title","customTitle":"自定义标题"}` + "\n"
	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	// Create handler with sessionTitle disabled
	pluginRoot := setupTestPlugin(t)
	handler, err := NewHandler(pluginRoot)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}

	// Disable session title extraction
	enabled := false
	handler.cfg.Notifications.Desktop.SessionTitle.Enabled = &enabled

	mockNotifier := &mockNotifier{}
	handler.notifierSvc = mockNotifier

	input := fmt.Sprintf(`{"transcript_path":%q,"session_id":"test-session","cwd":"%s"}`, transcriptPath, transcriptDir)

	if err := handler.HandleHook("Stop", strings.NewReader(input)); err != nil {
		t.Fatalf("HandleHook failed: %v", err)
	}

	if !mockNotifier.desktopCalled {
		t.Fatal("expected desktop notification to be called")
	}

	// Should use generated label, not custom title
	if strings.Contains(mockNotifier.lastMessage, "自定义标题") {
		t.Errorf("expected generated label, not custom title, got: %s", mockNotifier.lastMessage)
	}
}
```

注意：如果 `hooks_test.go` 中已有 `mockNotifier` 定义，复用现有结构。如果没有，需要添加：

```go
type mockNotifier struct {
	desktopCalled bool
	lastMessage   string
}

func (m *mockNotifier) SendDesktop(status analyzer.Status, message, sessionID, cwd string) error {
	m.desktopCalled = true
	m.lastMessage = message
	return nil
}

func (m *mockNotifier) Close() error {
	return nil
}
```

- [ ] **Step 2: 运行新增测试**

Run: `go test ./internal/hooks/... -run TestHandleHook_Stop_WithCustomTitle -v`
Expected: PASS

Run: `go test ./internal/hooks/... -run TestHandleHook_Stop_SessionTitleDisabled -v`
Expected: PASS

- [ ] **Step 3: 运行全部测试**

Run: `go test ./...`
Expected: PASS (所有包测试通过)

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/hooks_test.go
git commit -m "test(hooks): 添加会话标题提取集成测试"
```

---

### Task 6: 最终验证与提交

- [ ] **Step 1: 运行完整测试套件**

Run: `go test ./... -race`
Expected: PASS (无数据竞争)

- [ ] **Step 2: 构建二进制文件**

Run: `go build ./cmd/claude-notifications/...`
Expected: 编译成功

- [ ] **Step 3: 最终提交（如需要）**

如有未提交的变更：
```bash
git add -A && git commit -m "feat: 完成自定义会话标题功能"
```

---

## 自我审查清单

**1. Spec 覆盖检查：**

| Spec 需求 | 对应任务 |
|-----------|----------|
| 从 transcript 提取 customTitle | Task 2: ExtractTitle 处理 `type: "custom-title"` |
| 从 transcript 提取 aiTitle | Task 2: ExtractTitle 处理 `type: "ai-title"` |
| 从 transcript 提取 slug | Task 2: ExtractTitle 处理 `Slug` 字段 |
| 从 transcript 提取 firstUserPrompt | Task 2: ExtractTitle 处理 `type: "user"` |
| 优先级：custom > ai > slug > firstUserPrompt | Task 2: TestExtractTitle_Priority |
| 跳过 meta 用户消息 | Task 2: TestExtractTitle_SkipMetaUser |
| 跳过 `<` 开头的系统消息 | Task 2: TestExtractTitle_SkipSystemMessage |
| Unicode 安全截断 | Task 2: TruncateTitle 使用 []rune |
| 可配置 enabled | Task 3: SessionTitleConfig.Enabled |
| 可配置 maxLength | Task 3: SessionTitleConfig.MaxLength |
| 默认启用、默认长度 25 | Task 3: DefaultConfig 和 ApplyDefaults |
| 失败回退到 generated label | Task 4: 仅当 extracted != "" 时替换 |
| 仅 Stop hook 使用（有 messages） | Task 4: 其他 hook 传 nil |
| Windows 平台验证 | Task 5: 集成测试 + 完整测试套件 |

**2. 占位符检查：** 无 TBD、TODO、"implement later" 等占位符。

**3. 类型一致性检查：**
- `SessionTitleConfig` 在 config.go 中定义，与 hooks.go 中通过 `cfg.GetSessionTitleEnabled()` / `cfg.GetSessionTitleMaxLength()` 访问一致
- `sendNotifications` 签名中的 `messages []jsonl.Message` 在所有调用点一致
- `ExtractTitle` / `TruncateTitle` 的函数签名在实现和测试中一致
