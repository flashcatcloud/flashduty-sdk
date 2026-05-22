package flashduty

import "context"

// DataSourceIntegration represents a configured Flashduty integration data source.
type DataSourceIntegration struct {
	DataSourceID int64          `json:"data_source_id" toon:"data_source_id"`
	Name         string         `json:"name,omitempty" toon:"name,omitempty"`
	PluginType   string         `json:"plugin_type,omitempty" toon:"plugin_type,omitempty"`
	Category     string         `json:"category,omitempty" toon:"category,omitempty"`
	Settings     map[string]any `json:"settings,omitempty" toon:"settings,omitempty"`
}

// ListWarRoomEnabledDataSourcesOutput contains IM integrations that have war-room enabled.
type ListWarRoomEnabledDataSourcesOutput struct {
	Items []DataSourceIntegration `json:"items" toon:"items"`
}

// ListWarRoomEnabledDataSources lists IM integrations with war-room creation enabled.
func (c *Client) ListWarRoomEnabledDataSources(ctx context.Context) (*ListWarRoomEnabledDataSourcesOutput, error) {
	return postData[ListWarRoomEnabledDataSourcesOutput](c, ctx, "/datasource/im/war-room-enabled/list", map[string]any{}, "failed to list war-room enabled data sources")
}
