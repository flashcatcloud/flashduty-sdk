package flashduty

import (
	"context"
	"encoding/json"
	"fmt"
)

// MonitAgentCatalogInput is the request payload for /monit/tools/catalog.
// TargetKind is optional — the agent infers it from TargetLocator when
// unambiguous (e.g. host:web-01 → host, mysql:prod-master → mysql).
type MonitAgentCatalogInput struct {
	TargetLocator string
	TargetKind    string
}

// MonitAgentTool describes a single tool the runner exposes for a target.
// InputSchema is a JSON Schema fragment, intentionally kept as RawMessage so
// callers can pipe it straight to a tool-using LLM without re-marshalling.
type MonitAgentTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// MonitAgentCatalogOutput is the decoded response from /monit/tools/catalog.
type MonitAgentCatalogOutput struct {
	Tools []MonitAgentTool `json:"tools"`
}

// MonitAgentCatalog lists the tools the runner exposes for the given target.
func (c *Client) MonitAgentCatalog(ctx context.Context, input *MonitAgentCatalogInput) (*MonitAgentCatalogOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("monit agent catalog: input is required")
	}

	requestBody := map[string]any{
		"target_locator": input.TargetLocator,
	}
	if input.TargetKind != "" {
		requestBody["target_kind"] = input.TargetKind
	}

	return postData[MonitAgentCatalogOutput](c, ctx, "/monit/tools/catalog", requestBody, "failed to list monit agent tool catalog")
}

// MonitAgentInvokeTool is a single entry in the /monit/tools/invoke `tools`
// array. Params is the tool-specific argument payload — left as RawMessage
// so callers can pass the JSON they already have without round-tripping
// through map[string]any.
type MonitAgentInvokeTool struct {
	Tool   string          `json:"tool"`
	Params json.RawMessage `json:"params,omitempty"`
}

// MonitAgentInvokeResult is one entry in the returned `results` array. The
// runner returns results in the request order, with the same length as the
// input `tools` array, even when some tools error — callers must inspect
// Error per result and not assume "no outer error" means all tools succeeded.
//
// Data carries the per-tool payload (typically {summary, data}) and is left
// as RawMessage so the agent doesn't have to model every tool's shape.
type MonitAgentInvokeResult struct {
	Tool  string          `json:"tool"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// MonitAgentInvokeInput is the request payload for /monit/tools/invoke.
// At most 8 tools may be invoked per call; the runner executes them
// concurrently and returns them in the same order.
type MonitAgentInvokeInput struct {
	TargetLocator string
	TargetKind    string
	Tools         []MonitAgentInvokeTool
}

// MonitAgentInvokeOutput is the decoded response from /monit/tools/invoke.
//
// Three error layers exist and callers must distinguish them:
//  1. The error returned by this method — an HTTP-level failure
//     (network error, 5xx, malformed JSON).
//  2. A request-level error wrapped in dataEnvelope.Error — surfaced as the
//     method's returned error (target_unavailable, ambiguous_target_kind,
//     unknown_toolset_hash, forward_failed). When this fires, Results is
//     not populated.
//  3. Results[i].Error — a per-tool failure; other entries in Results may
//     have succeeded and callers should consume them.
type MonitAgentInvokeOutput struct {
	Results []MonitAgentInvokeResult `json:"results"`
}

// MonitAgentInvoke runs one or more tools against the given target. Tools
// execute concurrently on the runner; the response preserves request order.
func (c *Client) MonitAgentInvoke(ctx context.Context, input *MonitAgentInvokeInput) (*MonitAgentInvokeOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("monit agent invoke: input is required")
	}

	requestBody := map[string]any{
		"target_locator": input.TargetLocator,
		"tools":          input.Tools,
	}
	if input.TargetKind != "" {
		requestBody["target_kind"] = input.TargetKind
	}

	return postData[MonitAgentInvokeOutput](c, ctx, "/monit/tools/invoke", requestBody, "failed to invoke monit agent tools")
}
