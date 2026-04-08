# flashduty-sdk

Go SDK for the [FlashDuty](https://flashcat.cloud) API. Provides typed methods for incident management, on-call scheduling, status pages, notification templates, and more.

## Installation

```bash
go get github.com/flashcatcloud/flashduty-sdk
```

Requires Go 1.24+.

## Quick Start

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

## Client Options

```go
client, err := flashduty.NewClient("your-app-key",
	flashduty.WithBaseURL("https://custom-api.example.com"),
	flashduty.WithTimeout(10 * time.Second),
	flashduty.WithUserAgent("my-app/1.0"),
	flashduty.WithHTTPClient(customHTTPClient),
	flashduty.WithLogger(myLogger),
	flashduty.WithRequestHeaders(staticHeaders),
	flashduty.WithRequestHook(func(req *http.Request) {
		// Inject per-request headers (e.g., W3C Trace Context)
		req.Header.Set("traceparent", traceID)
	}),
)
```

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL` | `https://api.flashcat.cloud` | API base URL |
| `WithTimeout` | `30s` | HTTP client timeout |
| `WithUserAgent` | `flashduty-go-sdk` | User-Agent header |
| `WithHTTPClient` | Default `http.Client` | Custom HTTP client |
| `WithLogger` | `slog`-based logger | Custom logger implementing `Logger` interface |
| `WithRequestHeaders` | none | Static headers included in every request |
| `WithRequestHook` | none | Callback invoked on every outgoing request before it is sent |

### Dynamic User-Agent

The User-Agent can be updated after client creation (e.g., per-session):

```go
client.SetUserAgent("my-app/2.0 (client-name/1.2)")
```

## Logger Interface

The SDK uses a pluggable logger. The default implementation wraps `log/slog`.

```go
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}
```

To adapt logrus or other backends:

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

## API Reference

### Incidents

```go
// List incidents by IDs or filters (time-based queries require StartTime and EndTime)
client.ListIncidents(ctx, &ListIncidentsInput{...}) (*ListIncidentsOutput, error)

// Get timeline events for one or more incidents
client.GetIncidentTimelines(ctx, incidentIDs) ([]IncidentTimelineOutput, error)

// Get alerts for one or more incidents
client.ListIncidentAlerts(ctx, incidentIDs, limit) ([]IncidentAlertsOutput, error)

// Find similar historical incidents
client.ListSimilarIncidents(ctx, incidentID, limit) (*ListIncidentsOutput, error)

// Create a new incident
client.CreateIncident(ctx, &CreateIncidentInput{...}) (any, error)

// Update incident fields (title, description, severity, custom fields)
client.UpdateIncident(ctx, &UpdateIncidentInput{...}) ([]string, error)

// Acknowledge incidents
client.AckIncidents(ctx, incidentIDs) error

// Close (resolve) incidents
client.CloseIncidents(ctx, incidentIDs) error
```

### Members

```go
// List members by person IDs, name, or email
client.ListMembers(ctx, &ListMembersInput{...}) (*ListMembersOutput, error)
```

### Teams

```go
// List teams by team IDs or name
client.ListTeams(ctx, &ListTeamsInput{...}) (*ListTeamsOutput, error)
```

### Channels (Collaboration Spaces)

```go
// List channels by IDs or name (name filtering is case-insensitive substring match)
client.ListChannels(ctx, &ListChannelsInput{...}) (*ListChannelsOutput, error)
```

### Escalation Rules

```go
// List escalation rules for a channel (enriched with person/team/schedule names)
client.ListEscalationRules(ctx, channelID) (*ListEscalationRulesOutput, error)
```

### Custom Fields

```go
// List custom field definitions, optionally filtered by IDs or name
client.ListFields(ctx, &ListFieldsInput{...}) (*ListFieldsOutput, error)
```

### Changes

```go
// List change records (deployments, configurations) with enriched names
client.ListChanges(ctx, &ListChangesInput{...}) (*ListChangesOutput, error)
```

### Status Pages

```go
// List status pages, optionally filtered by page IDs
client.ListStatusPages(ctx, pageIDs) ([]StatusPage, error)

// List active incidents or maintenances on a status page
client.ListStatusChanges(ctx, &ListStatusChangesInput{...}) (*ListStatusChangesOutput, error)

// Create an incident on a status page
client.CreateStatusIncident(ctx, &CreateStatusIncidentInput{...}) (any, error)

// Add a timeline update to a status page incident or maintenance
client.CreateChangeTimeline(ctx, &CreateChangeTimelineInput{...}) error
```

### Templates

```go
// Fetch the preset (default) notification template for a channel
client.GetPresetTemplate(ctx, &GetPresetTemplateInput{...}) (*GetPresetTemplateOutput, error)

// Validate and preview a notification template with size-limit checks
client.ValidateTemplate(ctx, &ValidateTemplateInput{...}) (*ValidateTemplateOutput, error)
```

#### Static Template Data

These package-level functions return compiled-in reference data for template authoring:

```go
// Available template variables (40 variables across 7 categories)
flashduty.TemplateVariables() []TemplateVariable

// Custom FlashDuty template functions (19 functions)
flashduty.TemplateCustomFunctions() []TemplateFunction

// Commonly used Sprig template functions (19 functions)
flashduty.TemplateSprigFunctions() []TemplateFunction

// Valid notification channel identifiers (13 channels)
flashduty.ChannelEnumValues() []string
```

Supported channels: `dingtalk`, `dingtalk_app`, `feishu`, `feishu_app`, `wecom`, `wecom_app`, `slack`, `slack_app`, `telegram`, `teams_app`, `email`, `sms`, `zoom`.

Channel size limits and channel-to-field mappings are available via `flashduty.ChannelSizeLimits` and `flashduty.TemplateChannels`.

> **Note:** Static template data is compiled into the SDK. Platform-side additions require an SDK release.

## Data Enrichment

Most query methods automatically enrich raw API data with human-readable names. For example, `ListIncidents` resolves `CreatorID` to `CreatorName`, `ChannelID` to `ChannelName`, and responder person IDs to names and emails.

Enrichment uses concurrent batch fetches via `errgroup`. For methods like `ListChanges` and `ListChannels`, enrichment failures are best-effort -- the primary data is still returned even if name resolution fails.

## Output Formats

The SDK supports JSON and [TOON](https://github.com/toon-format/toon-go) (Token-Oriented Object Notation) serialization:

```go
data, err := flashduty.Marshal(incidents, flashduty.OutputFormatJSON)
data, err := flashduty.Marshal(incidents, flashduty.OutputFormatTOON)

format := flashduty.ParseOutputFormat("toon") // defaults to JSON for unknown values
```

## Error Handling

API errors are returned as `*DutyError` which implements the `error` interface:

```go
incidents, err := client.ListIncidents(ctx, input)
if err != nil {
	var dutyErr *flashduty.DutyError
	if errors.As(err, &dutyErr) {
		fmt.Printf("API error [%s]: %s\n", dutyErr.Code, dutyErr.Message)
	}
}
```

## License

See [LICENSE](LICENSE) for details.
