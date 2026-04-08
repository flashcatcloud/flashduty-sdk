# flashduty-sdk

Go SDK for the [FlashDuty](https://flashcat.cloud) API. Provides typed methods for incident management, on-call scheduling, status pages, and more.

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
)
```

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL` | `https://api.flashcat.cloud` | API base URL |
| `WithTimeout` | `30s` | HTTP client timeout |
| `WithUserAgent` | `flashduty-go-sdk` | User-Agent header |
| `WithHTTPClient` | Default `http.Client` | Custom HTTP client |

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
