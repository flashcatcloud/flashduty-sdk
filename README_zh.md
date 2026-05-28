# Flashduty SDK

[English](README.md) | 中文

[![License](https://img.shields.io/github/license/flashcatcloud/flashduty-sdk?style=flat-square&color=24bfa5&label=License)](LICENSE)
[![Go Reference](https://img.shields.io/badge/Go-Reference-24bfa5?style=flat-square&logo=go)](https://pkg.go.dev/github.com/flashcatcloud/flashduty-sdk)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/flashduty-sdk/go.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/flashduty-sdk/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/flashcatcloud/flashduty-sdk?style=flat-square)](https://goreportcard.com/report/github.com/flashcatcloud/flashduty-sdk)

[Flashduty](https://flashcat.cloud) 平台 API 的 Go SDK。提供故障管理、值班排班、状态页、通知模板等能力的类型化方法。

## 安装

```bash
go get github.com/flashcatcloud/flashduty-sdk
```

需要 Go 1.24+。

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

func main() {
	client, err := flashduty.NewClient("your-app-key")
	if err != nil {
		log.Fatal(err)
	}

	incidents, err := client.ListIncidents(context.Background(), &flashduty.ListIncidentsInput{
		Progress:  "Triggered",
		StartTime: 1710000000,
		EndTime:   1710086400,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, inc := range incidents.Incidents {
		fmt.Printf("[%s] %s (channel: %s)\n", inc.Severity, inc.Title, inc.ChannelName)
	}
}
```

## 客户端选项

```go
client, err := flashduty.NewClient("your-app-key",
	flashduty.WithBaseURL("https://custom-api.example.com"),
	flashduty.WithTimeout(10 * time.Second),
	flashduty.WithUserAgent("my-app/1.0"),
	flashduty.WithHTTPClient(customHTTPClient),
	flashduty.WithLogger(myLogger),
	flashduty.WithRequestHeaders(staticHeaders),
	flashduty.WithRequestHook(func(req *http.Request) {
		// 注入按请求的请求头（例如 W3C Trace Context）
		req.Header.Set("traceparent", traceID)
	}),
)
```

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithBaseURL` | `https://api.flashcat.cloud` | API 地址 |
| `WithTimeout` | `30s` | HTTP 客户端超时 |
| `WithUserAgent` | `flashduty-go-sdk` | User-Agent 请求头 |
| `WithHTTPClient` | 默认 `http.Client` | 自定义 HTTP 客户端 |
| `WithLogger` | 基于 `slog` 的日志器 | 实现 `Logger` 接口的自定义日志器 |
| `WithRequestHeaders` | 无 | 每个请求都包含的静态请求头 |
| `WithRequestHook` | 无 | 每个出站请求发送前调用的回调 |

### 动态 User-Agent

创建客户端之后仍可更新 User-Agent（例如按会话）：

```go
client.SetUserAgent("my-app/2.0 (client-name/1.2)")
```

## 日志接口

SDK 使用可插拔的日志器，默认实现封装了 `log/slog`。

```go
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}
```

适配 logrus 或其他后端：

```go
type logrusAdapter struct{ *logrus.Logger }

func (a *logrusAdapter) Info(msg string, kv ...any)  { a.WithFields(kvToFields(kv)).Info(msg) }
func (a *logrusAdapter) Warn(msg string, kv ...any)  { a.WithFields(kvToFields(kv)).Warn(msg) }
func (a *logrusAdapter) Error(msg string, kv ...any) { a.WithFields(kvToFields(kv)).Error(msg) }
func (a *logrusAdapter) Debug(msg string, kv ...any) { a.WithFields(kvToFields(kv)).Debug(msg) }

func kvToFields(kv []any) logrus.Fields {
	fields := make(logrus.Fields, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		if key, ok := kv[i].(string); ok {
			fields[key] = kv[i+1]
		}
	}
	return fields
}
```

## API 参考

### 故障（Incidents）

```go
// 按 ID 或筛选条件列出故障（基于时间的查询需要 StartTime 和 EndTime）
client.ListIncidents(ctx, &ListIncidentsInput{...}) (*ListIncidentsOutput, error)

// 获取一个或多个故障的时间线事件
client.GetIncidentTimelines(ctx, incidentIDs) ([]IncidentTimelineOutput, error)

// 获取一个或多个故障的告警
client.ListIncidentAlerts(ctx, incidentIDs, limit) ([]IncidentAlertsOutput, error)

// 查找相似的历史故障
client.ListSimilarIncidents(ctx, incidentID, limit) (*ListIncidentsOutput, error)

// 创建新故障
client.CreateIncident(ctx, &CreateIncidentInput{...}) (any, error)

// 更新故障字段（标题、描述、级别、自定义字段）
client.UpdateIncident(ctx, &UpdateIncidentInput{...}) ([]string, error)

// 认领故障
client.AckIncidents(ctx, incidentIDs) error

// 关闭（解决）故障
client.CloseIncidents(ctx, incidentIDs) error
```

### 成员（Members）

```go
// 按人员 ID、姓名或邮箱列出成员
client.ListMembers(ctx, &ListMembersInput{...}) (*ListMembersOutput, error)
```

### 团队（Teams）

```go
// 按团队 ID 或名称列出团队
client.ListTeams(ctx, &ListTeamsInput{...}) (*ListTeamsOutput, error)
```

### 协作空间（Channels）

```go
// 按 ID 或名称列出协作空间（名称匹配为不区分大小写的子串匹配）
client.ListChannels(ctx, &ListChannelsInput{...}) (*ListChannelsOutput, error)
```

### 分派策略（Escalation Rules）

```go
// 列出某协作空间的分派策略（附带人员/团队/排班名称）
client.ListEscalationRules(ctx, channelID) (*ListEscalationRulesOutput, error)
```

### 自定义字段（Custom Fields）

```go
// 列出自定义字段定义，可按 ID 或名称筛选
client.ListFields(ctx, &ListFieldsInput{...}) (*ListFieldsOutput, error)
```

### 变更记录（Changes）

```go
// 列出变更记录（部署、配置等），附带解析后的名称
client.ListChanges(ctx, &ListChangesInput{...}) (*ListChangesOutput, error)
```

### 状态页（Status Pages）

```go
// 列出状态页，可按页面 ID 筛选
client.ListStatusPages(ctx, pageIDs) ([]StatusPage, error)

// 列出状态页上活跃的事件或维护
client.ListStatusChanges(ctx, &ListStatusChangesInput{...}) (*ListStatusChangesOutput, error)

// 在状态页上创建事件
client.CreateStatusIncident(ctx, &CreateStatusIncidentInput{...}) (any, error)

// 为状态页事件或维护添加时间线更新
client.CreateChangeTimeline(ctx, &CreateChangeTimelineInput{...}) error
```

### 通知模板（Templates）

```go
// 获取某渠道的预设（默认）通知模板
client.GetPresetTemplate(ctx, &GetPresetTemplateInput{...}) (*GetPresetTemplateOutput, error)

// 校验并预览通知模板，含大小限制检查
client.ValidateTemplate(ctx, &ValidateTemplateInput{...}) (*ValidateTemplateOutput, error)
```

#### 静态模板数据

以下包级函数返回编译进 SDK 的参考数据，用于编写模板：

```go
// 可用的模板变量（7 个分类共 40 个变量）
flashduty.TemplateVariables() []TemplateVariable

// Flashduty 自定义模板函数（19 个）
flashduty.TemplateCustomFunctions() []TemplateFunction

// 常用的 Sprig 模板函数（19 个）
flashduty.TemplateSprigFunctions() []TemplateFunction

// 合法的通知渠道标识（13 个渠道）
flashduty.ChannelEnumValues() []string
```

支持的通知渠道：`dingtalk`、`dingtalk_app`、`feishu`、`feishu_app`、`wecom`、`wecom_app`、`slack`、`slack_app`、`telegram`、`teams_app`、`email`、`sms`、`zoom`。

渠道大小限制与渠道-字段映射分别通过 `flashduty.ChannelSizeLimits` 和 `flashduty.TemplateChannels` 提供。

> **注意：** 静态模板数据编译在 SDK 内，平台侧新增项需要随 SDK 发版才能获取。

## 数据补全（Enrichment）

大多数查询方法会自动用可读名称补全原始 API 数据。例如 `ListIncidents` 会把 `CreatorID` 解析为 `CreatorName`、`ChannelID` 解析为 `ChannelName`、响应人的人员 ID 解析为姓名和邮箱。

补全通过 `errgroup` 并发批量拉取。对 `ListChanges`、`ListChannels` 等方法，补全采用尽力而为策略——即使名称解析失败，主数据仍会返回。

## 输出格式

SDK 支持 JSON 与 [TOON](https://github.com/toon-format/toon-go)（Token-Oriented Object Notation）序列化：

```go
data, err := flashduty.Marshal(incidents, flashduty.OutputFormatJSON)
data, err := flashduty.Marshal(incidents, flashduty.OutputFormatTOON)

format := flashduty.ParseOutputFormat("toon") // 未知值默认回退到 JSON
```

## 错误处理

API 错误以 `*DutyError` 返回，它实现了 `error` 接口：

```go
incidents, err := client.ListIncidents(ctx, input)
if err != nil {
	var dutyErr *flashduty.DutyError
	if errors.As(err, &dutyErr) {
		fmt.Printf("API error [%s]: %s\n", dutyErr.Code, dutyErr.Message)
	}
}
```

## 开发

需要 Go 1.24+ 和 [golangci-lint v2](https://golangci-lint.run/welcome/install/)。

```bash
go test -race ./...   # 运行测试（启用竞态检测）
golangci-lint run     # 运行代码检查
```

## 参与贡献

欢迎贡献代码！提交 Pull Request 前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，并遵守我们的[行为准则](CODE_OF_CONDUCT.md)。

- [报告缺陷或提交需求](https://github.com/flashcatcloud/flashduty-sdk/issues/new/choose)
- [获取帮助与支持](SUPPORT.md)
- [报告安全漏洞](SECURITY.md)

## 许可证

本项目基于 MIT 许可证开源 - 详见 [LICENSE](LICENSE) 文件。
