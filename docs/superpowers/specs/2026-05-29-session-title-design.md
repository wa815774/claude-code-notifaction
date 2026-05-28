# 自定义会话标题设计文档

## 背景

当前系统通知的标题使用 `sessionname.GenerateSessionLabel(sessionID)` 生成类似 `river 696e6596` 的友好名称。该名称由会话 ID 哈希推导，不包含任何语义信息，用户无法从通知标题判断是哪个会话完成了任务。

本设计引入从 Claude Code transcript 中提取语义化会话标题的能力，替代或增强现有的生成式标签。

## 目标

1. 通知标题显示有意义的会话标题（如 `"修改任务完成的系统通知"`）而非随机标签
2. 自动提取，无需用户手动配置每个会话的标题
3. 可配置开关和截断长度，具备生产级健壮性
4. 提取失败时无缝回退到现有行为

## 非目标

1. 不实现 `/rename` 命令（由 Claude Code 自身提供）
2. 不从终端窗口标题提取（hook 进程无控制 TTY，不可靠）
3. 不添加 AI 标题生成功能（由 Claude Code 自身提供）

## 方案概述

从 transcript JSONL 中提取以下字段（按优先级降序）：

| 优先级 | 字段 | 来源 | 说明 |
|--------|------|------|------|
| 1 | `customTitle` | `type: "custom-title"` entry | 用户通过 `/rename` 手动设置 |
| 2 | `aiTitle` | `type: "ai-title"` entry | Claude Code AI 自动生成 |
| 3 | `slug` | 任意 entry 的 `slug` 字段 | 会话 slug |
| 4 | `firstUserPrompt` | 首条 `type: "user"` 消息 | 用户的首条消息内容 |
| 5 | `generatedLabel` | `sessionname.GenerateSessionLabel` | 现有回退行为 |

## 配置设计

```go
// SessionTitleConfig 自定义会话标题配置
type SessionTitleConfig struct {
    Enabled   *bool `json:"enabled,omitempty"`   // 默认 true
    MaxLength *int  `json:"maxLength,omitempty"` // 默认 25
}
```

嵌入现有 `DesktopConfig`：

```go
type DesktopConfig struct {
    // ... 现有字段 ...
    SessionTitle *SessionTitleConfig `json:"sessionTitle,omitempty"`
}
```

JSON 配置示例：

```json
{
  "notifications": {
    "desktop": {
      "sessionTitle": {
        "enabled": true,
        "maxLength": 25
      }
    }
  }
}
```

省略时默认启用，`maxLength` 默认 25。

## 架构变更

### 新增模块

**`internal/sessiontitle/sessiontitle.go`**

```go
package sessiontitle

import "github.com/wa815774/claude-notifications/pkg/jsonl"

// ExtractTitle 从消息列表中提取会话标题
// 优先级：customTitle > aiTitle > slug > firstUserPrompt > ""
func ExtractTitle(messages []jsonl.Message) string

// TruncateTitle 截断标题到指定长度
// 使用 Unicode 安全截断（正确处理中文等多字节字符）
// 截断后追加 "…"
func TruncateTitle(title string, maxLen int) string
```

**`internal/sessiontitle/sessiontitle_test.go`**

覆盖场景：
- 各优先级字段独立存在时的提取
- 多个字段同时存在时的优先级选择
- Unicode 字符串截断（中、英、混合）
- 空消息列表回退
- 首条用户消息过滤（跳过 meta、以 `<` 开头的系统消息）

### 修改模块

**`internal/config/config.go`**

- `DesktopConfig` 新增 `SessionTitle *SessionTitleConfig` 字段
- `DefaultConfig()` 中默认启用，`maxLength` 默认 25
- 新增 getter 方法：`GetSessionTitleEnabled() bool`、`GetSessionTitleMaxLength() int`

**`internal/hooks/hooks.go`**

修改 `sendNotifications` 签名，增加 `messages` 参数：

```go
func (h *Handler) sendNotifications(status analyzer.Status, body, actions, sessionID, cwd string, messages []jsonl.Message) {
    // ... 现有代码 ...

    // 提取语义化会话标题
    sessionName := sessionname.GenerateSessionLabel(sessionID)
    if h.cfg.GetSessionTitleEnabled() && len(messages) > 0 {
        if extracted := sessiontitle.ExtractTitle(messages); extracted != "" {
            sessionName = sessiontitle.TruncateTitle(extracted, h.cfg.GetSessionTitleMaxLength())
        }
    }

    // ... 剩余逻辑不变 ...
}
```

对应修改 `HandleHook` 中的调用点：

```go
// 原调用
h.sendNotifications(status, body, actions, hookData.SessionID, hookData.CWD)
// 新调用（Stop/SubagentStop hook 有 messages，其他传 nil）
h.sendNotifications(status, body, actions, hookData.SessionID, hookData.CWD, parsedMessages)
```

关键决策：**在 `hooks.go` 层提取标题**，原因：
1. `handleStopEvent` 已解析 transcript 并返回 messages，避免重复 I/O
2. 标题决策与通知发送逻辑同属编排层，职责边界清晰
3. 便于在 `sendNotifications` 上下文中统一测试

**`pkg/jsonl/jsonl.go`**

扩展 `Message` 结构以支持 transcript 中的标题相关字段：

```go
type Message struct {
    // ... 现有字段 ...
    CustomTitle string `json:"customTitle,omitempty"`
    AITitle     string `json:"aiTitle,omitempty"`
    Slug        string `json:"slug,omitempty"`
    IsMeta      bool   `json:"isMeta,omitempty"`
}
```

注意：这些字段使用 `omitempty`，不影响现有解析性能。

## 数据流

```
┌─────────────────┐
│   Stop hook     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────────┐
│  analyzer.Analyze│────▶│  []jsonl.Message     │
│  TranscriptWith  │     │  (已解析的 transcript)│
│  Messages        │     └──────────┬───────────┘
└─────────────────┘                │
                                   ▼
                          ┌──────────────────────┐
                          │ sessiontitle.Extract  │
                          │ Title(messages)       │
                          └──────────┬───────────┘
                                     │
                                     ▼
                          ┌──────────────────────┐
                          │ sessiontitle.Truncate │
                          │ Title(title, maxLen)  │
                          └──────────┬───────────┘
                                     │
                                     ▼
                          ┌──────────────────────┐
                          │ hooks.sendNotifications│
                          │ (使用语义化标题)        │
                          └──────────┬───────────┘
                                     │
                                     ▼
                          ┌──────────────────────┐
                          │ notifier.SendDesktop  │
                          │ (title 参数携带标题)   │
                          └──────────────────────┘
```

## 错误处理

| 场景 | 行为 |
|------|------|
| transcript 解析失败 | 跳过标题提取，使用 generated label |
| 所有字段均为空 | 使用 generated label |
| 标题长度超过 maxLength | Unicode 安全截断，追加 "…" |
| 配置 `enabled: false` | 完全跳过提取，保持现有行为 |
| 非 Stop hook（无 messages）| 使用 generated label（仅 Stop hook 有解析后的 messages）|

## 测试策略

### 单元测试

**`internal/sessiontitle/sessiontitle_test.go`**

| 用例 | 输入 | 期望输出 |
|------|------|---------|
| 仅 customTitle | `[{Type:"custom-title", CustomTitle:"My Title"}]` | `"My Title"` |
| customTitle + aiTitle | 同上，附加 aiTitle | `"My Title"`（custom 优先）|
| 仅 aiTitle | `[{Type:"ai-title", AITitle:"AI Title"}]` | `"AI Title"` |
| 仅 slug | `[{Slug:"my-slug"}]` | `"my-slug"` |
| 仅 firstUserPrompt | 首条 user 消息 | `"用户消息内容"` |
| 空列表 | `[]` | `""` |
| 仅 meta user 消息 | `isMeta: true` 的 user | `""`（跳过 meta）|
| Unicode 截断 | `"使用playwright操作chrome浏览器"`, maxLen: 10 | `"使用playwright操…"` |
| 英文截断 | `"Configure Playwright MCP"`, maxLen: 10 | `"Configure P…"` |

### 集成测试

**`internal/hooks/hooks_test.go`**

- 在现有 Stop hook 测试中，验证当 transcript 包含 customTitle 时，通知消息中的 sessionName 为提取的标题
- 验证配置 `enabled: false` 时，仍使用 generated label

### 回归测试

- 现有通知测试不修改断言（默认启用提取，但无标题字段时行为不变）
- 验证 Windows 平台通知格式正确（标题提取为纯数据层逻辑，与平台无关；非 Windows 平台通知验证不在本次范围）

## 回滚计划

1. 用户可通过配置 `"sessionTitle": { "enabled": false }` 立即禁用
2. 若存在严重 bug，回退到上一个版本的二进制文件即可（无数据迁移）

## 验收标准

- [ ] `customTitle`、`aiTitle`、`slug`、`firstUserPrompt` 均可从 transcript 正确提取
- [ ] 标题按正确优先级选择
- [ ] 标题超长时 Unicode 安全截断
- [ ] 配置 `enabled: false` 时完全保持现有行为
- [ ] 所有新增代码覆盖率 ≥ 90%
- [ ] 现有集成测试全部通过
- [ ] Windows 通知中文标题无乱码（复用现有 UTF-16LE 编码方案；非 Windows 平台验证不在本次范围）
